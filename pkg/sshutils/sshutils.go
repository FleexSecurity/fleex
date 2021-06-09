package sshutils

import (
	"io/ioutil"
	"log"
	"os/user"
	"path"
	"strings"
)

func GetLocalPublicSSHKey() string {
	rawKey := fileToString(path.Join(getHomeDir(), ".ssh", "id_ed25519.pub"))
	retString := strings.ReplaceAll(rawKey, "\r\n", "")
	retString = strings.ReplaceAll(retString, "\n", "")

	return retString
}

func fileToString(filePath string) string {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}

	return string(content)
}

func getHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}
