package controller

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/sirupsen/logrus"
	"golang.org/x/term"

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
	case PROVIDER_LINODE:
		c.Service = services.LinodeService{
			Client:  config.GetLinodeClient(token),
			Configs: configs,
		}
	case PROVIDER_DIGITALOCEAN:
		c.Service = services.DigitaloceanService{
			Client: config.GetDigitaloaceanToken(token),
		}
	case PROVIDER_VULTR:
		c.Service = services.VultrService{
			Client: config.GetVultrClient(token),
		}
	default:
		utils.Log.Fatal(provider.ErrInvalidProvider)
	}

	return c
}

// ListBoxes prints all active boxes of a provider
func (c Controller) ListBoxes(token string, provider Provider) {
	c.Service.ListBoxes()
}

// DeleteFleet deletes a whole fleet or a single box
func (c Controller) DeleteFleet(name string, token string, provider Provider) {
	err := c.Service.DeleteFleet(name)
	if err != nil {
		utils.Log.Fatal(err)
	}

	time.Sleep(1 * time.Second)
	for len(c.GetFleet(name, token, provider)) > 0 {
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

func (c Controller) GetFleet(fleetName string, token string, provider Provider) []provider.Box {
	fleet, err := c.Service.GetFleet(fleetName)
	if err != nil {
		utils.Log.Fatal(err)
	}
	return fleet
}

func (c Controller) GetBox(boxName string, token string, provider Provider) (provider.Box, error) {
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

func (c Controller) SpawnFleet(fleetName, password string, fleetCount int, image string, region string, size string, sshFingerprint string, tags []string, token string, skipWait bool, provider Provider, build bool) {
	startFleet := c.GetFleet(fleetName, token, provider)
	finalFleetSize := len(startFleet) + fleetCount

	if len(startFleet) > 0 {
		utils.Log.Info("Increasing fleet ", fleetName, " from size ", len(startFleet), " to ", finalFleetSize)
	}

	// Handle CTRL+C SIGINT
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		for range ch {
			utils.Log.Info("Spawn interrupted. Killing boxes...")
			c.DeleteFleet(fleetName, token, provider)
			os.Exit(0)
		}
	}()

	c.Service.SpawnFleet(fleetName, password, fleetCount, image, region, size, sshFingerprint, tags)

	if !skipWait {
		utils.Log.Info("All spawn requests sent! Now waiting for all boxes to become ready")
		for {
			stillNotReady := false
			fleet := c.GetFleet(fleetName, token, provider)
			if len(fleet) == finalFleetSize {
				for i := range fleet {
					if (provider == PROVIDER_DIGITALOCEAN && fleet[i].Status != "active") || (provider == PROVIDER_LINODE && fleet[i].Status != "running") || (provider == PROVIDER_VULTR && fleet[i].Status != "active") {
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

func (c Controller) SSH(boxName, username string, port int, sshKey string, token string, provider Provider) {
	box, err := c.GetBox(boxName, token, provider)
	if err != nil {
		utils.Log.Fatal(err)
	}

	if box.Label == boxName {
		c := exec.Command("ssh", "-i", "~/.ssh/"+sshKey, username+"@"+box.IP, "-p", strconv.Itoa(port))

		// Start the command with a pty.
		ptmx, err := pty.Start(c)
		if err != nil {
			utils.Log.Fatal(err)
		}
		// Make sure to close the pty at the end.
		defer func() { _ = ptmx.Close() }() // Best effort.

		// Handle pty size.
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGWINCH)
		go func() {
			for range ch {
				if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
					log.Printf("error resizing pty: %s", err)
				}
			}
		}()

		ch <- syscall.SIGWINCH                        // Initial resize.
		defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

		// Set stdin in raw mode.
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			utils.Log.Fatal(err)
		}
		defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

		// Copy stdin to the pty and the pty to stdout.
		// NOTE: The goroutine will keep reading until the next keystroke before returning.
		go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
		_, _ = io.Copy(os.Stdout, ptmx)

		return
	}
}

func SendSCP(source string, destination string, IP string, PORT int, username string) {
	// We were using a nice native SCP golang library but apparently they all suck so to avoid problems we do it this way

	utils.RunCommand("scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -P "+strconv.Itoa(PORT)+" "+source+" "+username+"@"+IP+":"+destination, false)
}
