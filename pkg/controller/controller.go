package controller

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	"github.com/FleexSecurity/fleex/config"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/provider"
	"github.com/FleexSecurity/fleex/pkg/services"
	"github.com/FleexSecurity/fleex/pkg/utils"
)

type Provider int

const (
	PROVIDER_CUSTOM       = 0
	PROVIDER_LINODE       = 1
	PROVIDER_DIGITALOCEAN = 2
	PROVIDER_VULTR        = 3
)

var log = logrus.New()

type Controller struct {
	Service provider.Provider
	Configs *models.Config
}

func GetProvider(name string) Provider {
	name = strings.ToLower(name)

	switch name {
	case "custom":
		return PROVIDER_CUSTOM
	case "linode":
		return PROVIDER_LINODE
	case "digitalocean":
		return PROVIDER_DIGITALOCEAN
	case "vultr":
		return PROVIDER_VULTR
	}

	return -1
}

func NewController(configs *models.Config) Controller {
	c := Controller{
		Configs: configs,
	}
	selectedProvider := configs.Settings.Provider
	providerId := GetProvider(selectedProvider)
	token := configs.Providers[selectedProvider].Token

	switch providerId {
	case PROVIDER_CUSTOM:
		c.Service = services.CustomService{
			Configs: configs,
		}
	case PROVIDER_LINODE:
		c.Service = services.LinodeService{
			Client:  config.GetLinodeClient(token),
			Configs: configs,
		}
	// case PROVIDER_DIGITALOCEAN:
	// 	c.Service = services.DigitaloceanService{
	// 		Client: config.GetDigitaloaceanToken(token),
	// 	}
	// case PROVIDER_VULTR:
	// 	c.Service = services.VultrService{
	// 		Client: config.GetVultrClient(token),
	// 	}
	default:
		utils.Log.Fatal(models.ErrInvalidProvider)
	}

	return c
}

// ListBoxes prints all active boxes of a provider
func (c Controller) ListBoxes(token string, provider Provider) {
	boxes, err := c.Service.GetBoxes()
	if err != nil {
		log.Fatal(err)
	}
	for _, linode := range boxes {
		fmt.Printf("%-20v %-16v %-10v %-20v %-15v\n", linode.ID, linode.Label, linode.Group, linode.Status, linode.IP)
	}
}

// DeleteFleet deletes a whole fleet or a single box
func (c Controller) DeleteFleet(name string) {
	err := c.Service.DeleteFleet(name)
	if err != nil {
		utils.Log.Fatal(err)
	}

	time.Sleep(1 * time.Second)
	for len(c.GetFleet(name)) > 0 {
		time.Sleep(1 * time.Second)
	}
	utils.Log.Info("Fleet/Box deleted!")
}

// ListImages prints a list of available private images of a provider
func (c Controller) ListImages(token string, provider Provider) {
	err := c.Service.ListImages()
	if err != nil {
		utils.Log.Fatal(err)
	}
}

func (c Controller) RemoveImages(token string, provider Provider, name string) {
	err := c.Service.RemoveImages(name)
	if err != nil {
		utils.Log.Fatal(err)
	}
}

func (c Controller) CreateImage(token string, provider Provider, diskID string, label string) {
	diskIDInt, _ := strconv.Atoi(diskID)
	err := c.Service.CreateImage(diskIDInt, label)
	if err != nil {
		utils.Log.Fatal(err)
	}
}

func (c Controller) GetFleet(fleetName string) []provider.Box {
	fleet, err := c.Service.GetFleet(fleetName)
	if err != nil {
		utils.Log.Fatal(err)
	}
	return fleet
}

func (c Controller) GetBox(boxName string) (provider.Box, error) {
	return c.Service.GetBox(boxName)
}

func (c Controller) RunCommand(name, command string) {
	provider := c.Configs.Settings.Provider
	port := c.Configs.Providers[provider].Port
	username := c.Configs.Providers[provider].Username
	password := c.Configs.Providers[provider].Password
	err := c.Service.RunCommand(name, command, port, username, password)
	if err != nil {
		utils.Log.Fatal(err)
	}
}

func (c Controller) DeleteBoxByID(id string, token string, provider Provider) {
	err := c.Service.DeleteBoxByID(id)
	if err != nil {
		utils.Log.Fatal(err)
	}
}

func (c Controller) SpawnFleet(fleetName string, fleetCount int, skipWait bool, build bool) {
	startFleet := c.GetFleet(fleetName)
	finalFleetSize := len(startFleet) + fleetCount
	selectedProvider := c.Configs.Settings.Provider
	providerId := GetProvider(selectedProvider)

	if len(startFleet) > 0 {
		utils.Log.Info("Increasing fleet ", fleetName, " from size ", len(startFleet), " to ", finalFleetSize)
	}

	// Handle CTRL+C SIGINT
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		for range ch {
			utils.Log.Info("Spawn interrupted. Killing boxes...")
			c.DeleteFleet(fleetName)
			os.Exit(0)
		}
	}()

	err := c.Service.SpawnFleet(fleetName, fleetCount)
	if err != nil {
		utils.Log.Fatal(err)
	}

	if !skipWait {
		utils.Log.Info("All spawn requests sent! Now waiting for all boxes to become ready")
		for {
			stillNotReady := false
			fleet := c.GetFleet(fleetName)
			if len(fleet) == finalFleetSize {
				for i := range fleet {
					if (providerId == PROVIDER_DIGITALOCEAN && fleet[i].Status != "active") || (providerId == PROVIDER_LINODE && fleet[i].Status != "running") || (providerId == PROVIDER_VULTR && fleet[i].Status != "active") {
						stillNotReady = true
					}
				}

				if stillNotReady {
					time.Sleep(8 * time.Second)
				} else {
					break
				}
			}

		}

		utils.Log.Info("All boxes ready!")

	}
}

func (c Controller) SSH(boxName, username, password string, port int, sshKey string) {
	box, err := c.GetBox(boxName)
	if err != nil {
		utils.Log.Fatal(err)
	}

	if box.Label == boxName {
		// key, err := sshutils.GetKey(sshKey)
		// if err != nil {
		// 	utils.Log.Fatal(err)
		// }

		config := &ssh.ClientConfig{
			User: username,
			Auth: []ssh.AuthMethod{
				ssh.Password("debian"),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

		addr := fmt.Sprintf("%s:%d", box.IP, port)
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			utils.Log.Fatal(err)
		}
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			utils.Log.Fatal(err)
		}
		defer session.Close()

		session.Stdout = os.Stdout
		session.Stderr = os.Stderr
		session.Stdin = os.Stdin

		if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
			modes := ssh.TerminalModes{
				ssh.ECHO:          1,
				ssh.TTY_OP_ISPEED: 14400,
				ssh.TTY_OP_OSPEED: 14400,
			}

			if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
				utils.Log.Fatal(err)
			}
		}

		err = session.Shell()
		if err != nil {
			utils.Log.Fatal(err)
		}

		err = session.Wait()
		if err != nil {
			utils.Log.Fatal(err)
		}
	}
}

func SendSCP(source, destination, ip, username string, port int, privateKeyPath string) error {
	key, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", ip+":"+strconv.Itoa(port), config)
	if err != nil {
		return err
	}
	defer client.Close()

	data, err := ioutil.ReadFile(source)
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()

		content := string(data)
		fmt.Fprintf(w, "C0644 %d %s\n", len(content), path.Base(destination))
		fmt.Fprint(w, content)
		fmt.Fprint(w, "\x00")
	}()

	if err := session.Run("/usr/bin/scp -tr " + path.Dir(destination)); err != nil {
		return err
	}

	return nil
}
