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
	"github.com/spf13/viper"
	"golang.org/x/term"

	"github.com/FleexSecurity/fleex/config"
	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/FleexSecurity/fleex/provider"
	"github.com/FleexSecurity/fleex/provider/services"
)

type Provider int

const (
	PROVIDER_LINODE       = 1
	PROVIDER_DIGITALOCEAN = 2
	PROVIDER_VULTR        = 3
	PROVIDER_CUSTOM       = 4
)

var log = logrus.New()

type Controller struct {
	Service provider.Provider
}

func GetProvider(name string) Provider {
	name = strings.ToLower(name)

	switch name {
	case "linode":
		return PROVIDER_LINODE
	case "digitalocean":
		return PROVIDER_DIGITALOCEAN
	case "vultr":
		return PROVIDER_VULTR
	case "custom":
		return PROVIDER_CUSTOM
	}

	return -1
}

func GetProviderController(pvd Provider, token string) Controller {
	c := Controller{}

	switch pvd {
	case PROVIDER_LINODE:
		c.Service = services.LinodeService{
			Client: config.GetLinodeClient(token),
		}
	case PROVIDER_DIGITALOCEAN:
		c.Service = services.DigitaloceanService{}
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
func ListBoxes(token string, provider Provider) {
	c := GetProviderController(provider, token)
	c.Service.ListBoxes(token)
}

// DeleteFleet deletes a whole fleet or a single box
func DeleteFleet(name string, token string, provider Provider) {
	c := GetProviderController(provider, token)
	err := c.Service.DeleteFleet(name, token)
	if err != nil {
		utils.Log.Fatal(err)
	}

	time.Sleep(1 * time.Second)
	for len(GetFleet(name, token, provider)) > 0 {
		time.Sleep(1 * time.Second)
	}
	utils.Log.Info("Fleet/Box deleted!")
}

// ListImages prints a list of available private images of a provider
func ListImages(token string, provider Provider) {
	c := GetProviderController(provider, token)
	err := c.Service.ListImages(token)
	if err != nil {
		utils.Log.Fatal(err)
	}
}

func CreateImage(token string, provider Provider, diskID string, label string) {
	c := GetProviderController(provider, token)
	diskIDInt, _ := strconv.Atoi(diskID)
	err := c.Service.CreateImage(token, diskIDInt, label)
	if err != nil {
		utils.Log.Fatal(err)
	}
}

func GetFleet(fleetName string, token string, provider Provider) []provider.Box {
	c := GetProviderController(provider, token)
	fleet, err := c.Service.GetFleet(fleetName, token)
	if err != nil {
		utils.Log.Fatal(err)
	}
	return fleet
}

func GetBox(boxName string, token string, provider Provider) (provider.Box, error) {
	c := GetProviderController(provider, token)
	return c.Service.GetBox(boxName, token)
}

func RunCommand(name, command, token string, port int, username, password string, providerNumber Provider) {
	if providerNumber == 4 {
		vmdata := viper.GetStringMapString("custom." + name)
		// sshutils.RunCommandWithPassword(command)
		if len(vmdata) == 0 {
			utils.Log.Fatal(provider.ErrInvalidProvider)
		}
		vmport, err := strconv.Atoi(vmdata["port"])
		if err != nil {
			utils.Log.Fatal(provider.ErrInvalidProvider)
		}
		sshutils.RunCommandWithPassword(command, vmdata["ip"], vmport, vmdata["username"], vmdata["password"])
		return
	}
	c := GetProviderController(providerNumber, token)
	err := c.Service.RunCommand(name, command, port, username, password, token)
	if err != nil {
		utils.Log.Fatal(err)
	}
}

func DeleteBoxByID(id string, token string, provider Provider) {
	c := GetProviderController(provider, token)
	err := c.Service.DeleteBoxByID(id, token)
	if err != nil {
		utils.Log.Fatal(err)
	}
}

func SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, sshFingerprint string, tags []string, token string, skipWait bool, provider Provider, build bool) {
	controller := GetProviderController(provider, token)
	startFleet := GetFleet(fleetName, token, provider)
	finalFleetSize := len(startFleet) + fleetCount

	if len(startFleet) > 0 {
		utils.Log.Info("Increasing fleet ", fleetName, " from size ", len(startFleet), " to ", finalFleetSize)
	}

	// Handle CTRL+C SIGINT
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			utils.Log.Info("Spawn interrupted. Killing boxes...")
			DeleteFleet(fleetName, token, provider)
			os.Exit(0)
		}
	}()

	controller.Service.SpawnFleet(fleetName, fleetCount, image, region, size, sshFingerprint, tags, token)

	if !skipWait {
		utils.Log.Info("All spawn requests sent! Now waiting for all boxes to become ready")
		for {
			stillNotReady := false
			fleet := GetFleet(fleetName, token, provider)
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

func SSH(boxName, username string, port int, sshKey string, token string, provider Provider) {
	box, err := GetBox(boxName, token, provider)
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
