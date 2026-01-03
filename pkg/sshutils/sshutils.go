package sshutils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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
	rawKey := utils.FileToString(publicSsh)
	retString := strings.ReplaceAll(rawKey, "\r\n", "")
	retString = strings.ReplaceAll(retString, "\n", "")

	return retString
}

func SSHFingerprintGen(publicSSH string) string {
	keyPath := publicSSH
	if !filepath.IsAbs(publicSSH) {
		keyPath = filepath.Join(getHomeDir(), ".ssh", publicSSH)
	}
	rawKey := utils.FileToString(keyPath)

	// Parse the key, other info ignored
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(rawKey))
	if err != nil {
		utils.Log.Fatal("SSHFingerprintGen: ", err)
	}

	// Get the fingerprint
	f := ssh.FingerprintLegacyMD5(pk)
	return f
}

func RunCommand(command string, ip string, port int, username string, privateKey string) *Connection {
	var conn *Connection
	var err error
	for retries := 0; retries < 3; retries++ {
		conn, err = Connect(ip+":"+strconv.Itoa(port), username, privateKey)
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

func RunCommandSilent(command string, ip string, port int, username string, privateKey string) (*Connection, error) {
	conn, err := Connect(ip+":"+strconv.Itoa(port), username, privateKey)
	if err != nil {
		return nil, err
	}

	_, err = conn.sendCommandsSilent(command)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func RunCommandWithOutput(command string, ip string, port int, username string, privateKey string) ([]byte, error) {
	conn, err := Connect(ip+":"+strconv.Itoa(port), username, privateKey)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	output, err := conn.sendCommandsSilent(command)
	return output, err
}

var termCount int

func (conn *Connection) sendCommands(cmds ...string) ([]byte, error) {
	session, err := conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("sendCommands: %w", err)
	}
	defer session.Close()

	if err := conn.setupPty(session); err != nil {
		return nil, err
	}

	stdin, stdout, stderr, err := setupStdPipes(session)
	if err != nil {
		return nil, err
	}
	defer stdin.Close()

	go io.Copy(stdin, os.Stdin)
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	cmd := strings.Join(cmds, "; ")
	output, err := session.Output(cmd)
	if err != nil {
		utils.Log.Errorf("Failed to execute command: %s, error: %v", cmd, err)
	}

	return output, nil
}

func (conn *Connection) sendCommandsSilent(cmds ...string) ([]byte, error) {
	session, err := conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("sendCommandsSilent: %w", err)
	}
	defer session.Close()

	cmd := strings.Join(cmds, "; ")
	output, err := session.CombinedOutput(cmd)

	return output, err
}

func (conn *Connection) setupPty(session *ssh.Session) error {
	term := os.Getenv("TERM")
	if term == "" {
		term = "xterm"
	}

	fd := int(os.Stdin.Fd())
	if termCount == 0 { // Assuming termCount's usage is justified and managed properly
		state, err := terminal.MakeRaw(fd)
		if err != nil {
			return fmt.Errorf("setupPty: making terminal raw: %w", err)
		}
		defer terminal.Restore(fd, state)
		termCount++
	}

	width, height, err := terminal.GetSize(fd)
	if err != nil {
		return fmt.Errorf("setupPty: getting terminal size: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		ssh.OPOST:         1,
	}

	if err := session.RequestPty(term, width, height, modes); err != nil {
		return fmt.Errorf("setupPty: requesting PTY: %w", err)
	}

	return nil
}

func setupStdPipes(session *ssh.Session) (io.WriteCloser, io.Reader, io.Reader, error) {
	stdin, err := session.StdinPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("setupStdPipes: stdin pipe setup failed: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("setupStdPipes: stdout pipe setup failed: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("setupStdPipes: stderr pipe setup failed: %w", err)
	}

	return stdin, stdout, stderr, nil
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

func Connect(addr, username, sshKey string) (*Connection, error) {
	key, err := ioutil.ReadFile(sshKey)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", addr, config)
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

// Generate Key Pair
func GenerateSSHKeyPair(bits int, email, path string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return err
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	privateKeyFile, err := os.Create(path + "/id_rsa")
	if err != nil {
		return err
	}
	defer privateKeyFile.Close()

	err = pem.Encode(privateKeyFile, privateKeyPEM)
	if err != nil {
		return err
	}

	publicKey, err := sshPublicKeyFromPrivateKey(privateKey, email)
	if err != nil {
		return err
	}

	publicKeyFile, err := os.Create(path + "/id_rsa.pub")
	if err != nil {
		return err
	}
	defer publicKeyFile.Close()

	_, err = publicKeyFile.WriteString(publicKey)
	if err != nil {
		return err
	}

	return nil
}

func sshPublicKeyFromPrivateKey(privateKey *rsa.PrivateKey, email string) (string, error) {
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", err
	}

	pubStr := strings.ReplaceAll(string(ssh.MarshalAuthorizedKey(pub)), "\n", "")
	email = strings.TrimSpace(email)

	return fmt.Sprintf("%s %s", pubStr, email), nil
}
