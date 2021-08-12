package sshutils

import (
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/utils"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

type Connection struct {
	*ssh.Client
}

func GetLocalPublicSSHKey() string {
	publicSsh := viper.GetString("public-ssh-file")
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
		panic(err)
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
			if strings.Contains(err.Error(), "connection refused") {
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
		utils.Log.Fatal(err)
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

func GetConnection(ip string, port int, username string, password string) *Connection {
	conn, err := Connect(ip+":"+strconv.Itoa(port), username, password)
	if err != nil {
		utils.Log.Fatal(err)
	}
	return conn
}

func GetConnectionBuild(ip string, port int, username string, password string) (*Connection, error) {
	conn, err := Connect(ip+":"+strconv.Itoa(port), username, password)
	return conn, err
}

func Connect(addr, user, password string) (*Connection, error) {
	privateSsh := viper.GetString("private-ssh-file")
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

func getHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		utils.Log.Fatal(err)
	}
	return usr.HomeDir
}
