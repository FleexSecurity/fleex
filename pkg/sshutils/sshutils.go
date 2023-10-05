package sshutils

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/utils"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

type Connection struct {
	*ssh.Client
}

func GetConfigs() *models.Config {
	configDir, err := utils.GetConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	cfgFile := filepath.Join(configDir, "fleex", "config.json")
	file, err := os.Open(cfgFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var config models.Config
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		log.Fatal(err)
	}

	return &config
}

func GetLocalPublicSSHKey() string {
	configs := GetConfigs()
	publicSsh := configs.SSHKeys.PublicFile
	rawKey := utils.FileToString(filepath.Join(getHomeDir(), ".ssh", publicSsh))
	retString := strings.ReplaceAll(rawKey, "\r\n", "")
	retString = strings.ReplaceAll(retString, "\n", "")

	return retString
}

func SSHFingerprintGen(publicSSH string) string {
	rawKey := utils.FileToString(filepath.Join(getHomeDir(), ".ssh", publicSSH))

	// Parse the key, other info ignored
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(rawKey))
	if err != nil {
		utils.Log.Fatal("SSHFingerprintGen: ", err)
	}

	// Get the fingerprint
	f := ssh.FingerprintLegacyMD5(pk)
	return f
}

func RunCommand(command string, ip string, port int, username string, password string) *Connection {
	var conn *Connection
	var err error
	for retries := 0; retries < 3; retries++ {
		conn, err = Connect(ip+":"+strconv.Itoa(port), username, password)
		if err != nil {
			if strings.Contains(err.Error(), "connection refused") && retries < 3 {
				continue
			}
			utils.Log.Fatal("RunCommand: ", err)
		}
		break
	}
	conn.sendCommands(command)

	return conn
}

func RunCommandWithPassword(command string, ip string, port int, username string, password string) *Connection {
	var conn *Connection
	var err error
	for retries := 0; retries < 3; retries++ {
		conn, err = ConnectWithPassword(ip+":"+strconv.Itoa(port), username, password)
		if err != nil {
			if strings.Contains(err.Error(), "connection refused") && retries < 3 {
				continue
			}
			utils.Log.Fatal(err)
		}
		break
	}
	conn.sendCommands(command)

	return conn
}

func publicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

var termCount int

func (conn *Connection) sendCommands(cmds ...string) ([]byte, error) {
	session, err := conn.NewSession()
	if err != nil {
		utils.Log.Fatal("sendCommands: ", err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		ssh.OPOST:         1,
	}

	term := os.Getenv("TERM")
	if term == "" {
		term = "xterm"
	}

	fd := int(os.Stdin.Fd())
	if termCount == 0 {
		state, err := terminal.MakeRaw(fd)
		if err != nil {
			utils.Log.Fatal("terminal make raw:", err)
		}
		defer terminal.Restore(fd, state)
		termCount++
	}

	terminalWidth, terminalHeight, err := terminal.GetSize(fd)
	if err != nil {
		utils.Log.Fatal("terminal get size:", err)
	}

	err = session.RequestPty(term, terminalWidth, terminalHeight, modes)
	if err != nil {
		return []byte{}, err
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		utils.Log.Fatal("Unable to setup stdin for session: ", err)
	}
	go io.Copy(stdin, os.Stdin)

	stdout, err := session.StdoutPipe()
	if err != nil {
		utils.Log.Fatal("Unable to setup stdout for session: ", err)
	}
	go io.Copy(os.Stdout, stdout)

	stderr, err := session.StderrPipe()
	if err != nil {
		utils.Log.Fatal("Unable to setup stderr for session: ", err)
	}
	go io.Copy(os.Stderr, stderr)

	cmd := strings.Join(cmds, "; ")
	output, err := session.Output(cmd)
	if err != nil {
		// We ignore it as we print the remote stderr in our local terminal already
		//return output, fmt.Errorf("failed to execute command '%s' on server: %v", cmd, err)
	}

	return output, err
}

func GetConnection(ip string, port int, username string, password string) (*Connection, error) {
	conn, err := Connect(ip+":"+strconv.Itoa(port), username, password)
	if err != nil {
		return nil, fmt.Errorf("GetConnection: %v, IP: %s, Port: %d, Username: %s", err, ip, port, username)
	}
	return conn, nil
}

func GetConnectionBuild(ip string, port int, username string, password string) (*Connection, error) {
	conn, err := Connect(ip+":"+strconv.Itoa(port), username, password)
	return conn, err
}

func Connect(addr, user, password string) (*Connection, error) {
	configs := GetConfigs()
	privateSsh := configs.SSHKeys.PrivateFile
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			publicKeyFile(filepath.Join(getHomeDir(), ".ssh", privateSsh)), // todo replace with rsa
		},
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
	}

	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, err
	}
	return &Connection{conn}, nil

}

func ConnectWithPassword(addr, user, password string) (*Connection, error) {
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
		// TODO: set up a timeout
	}

	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, err
	}
	return &Connection{conn}, nil

}

func getHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		utils.Log.Fatal("getHomeDir: ", err)
	}
	return usr.HomeDir
}
