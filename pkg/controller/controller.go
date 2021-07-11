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

	"github.com/sw33tLie/fleex/pkg/box"
	"github.com/sw33tLie/fleex/pkg/digitalocean"
	"github.com/sw33tLie/fleex/pkg/linode"
	"github.com/sw33tLie/fleex/pkg/utils"
)

type Provider int

const (
	PROVIDER_LINODE       = 1
	PROVIDER_DIGITALOCEAN = 2
)

const (
	INVALID_PROVIDER = `Something went wrong, check that the data in the config.yaml is correct.`
)

var log = logrus.New()

func GetProvider(name string) Provider {
	name = strings.ToLower(name)

	switch name {
	case "linode":
		return PROVIDER_LINODE
	case "digitalocean":
		return PROVIDER_DIGITALOCEAN
	}

	return -1
}

// ListBoxes prints all active boxes of a provider
func ListBoxes(token string, provider Provider) {
	switch provider {
	case PROVIDER_LINODE:
		linode.ListBoxes(token)
	case PROVIDER_DIGITALOCEAN:
		digitalocean.ListBoxes(token)
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
	}
}

// DeleteFleet deletes a whole fleet or a single box
func DeleteFleet(name string, token string, provider Provider) {
	switch provider {
	case PROVIDER_LINODE:
		linode.DeleteFleet(name, token)
	case PROVIDER_DIGITALOCEAN:
		digitalocean.DeleteFleet(name, token)
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
	}
}

// ListImages prints a list of available private images of a provider
func ListImages(token string, provider Provider) {
	switch provider {
	case PROVIDER_LINODE:
		linode.ListImages(token)
	case PROVIDER_DIGITALOCEAN:
		digitalocean.ListImages(token)
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
	}
}

func CreateImage(token string, provider Provider, diskID int, label string) {
	switch provider {
	case PROVIDER_LINODE:
		linode.CreateImage(token, diskID, label)
	case PROVIDER_DIGITALOCEAN:
		digitalocean.CreateImage(token, diskID, label)
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
	}
}

func GetFleet(fleetName string, token string, provider Provider) []box.Box {
	switch provider {
	case PROVIDER_LINODE:
		return linode.GetFleet(fleetName, token)
	case PROVIDER_DIGITALOCEAN:
		return digitalocean.GetFleet(fleetName, token)
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
		return nil
	}
}

func GetBox(boxName string, token string, provider Provider) box.Box {
	switch provider {
	case PROVIDER_LINODE:
		return linode.GetBox(boxName, token)
	case PROVIDER_DIGITALOCEAN:
		return digitalocean.GetBox(boxName, token)
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
		return box.Box{}
	}
}

func RunCommand(name, command, token string, port int, username, password string, provider Provider) {
	switch provider {
	case PROVIDER_LINODE:
		linode.RunCommand(name, command, port, username, password, token)
	case PROVIDER_DIGITALOCEAN:
		digitalocean.RunCommand(name, command, port, username, password, token)
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
	}
}

func DeleteBoxByID(id int, token string, provider Provider) {
	switch provider {
	case PROVIDER_LINODE:
		linode.DeleteBoxByID(id, token)
	case PROVIDER_DIGITALOCEAN:
		digitalocean.DeleteBoxByID(id, token)
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
	}
}

func SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, sshFingerprint string, tags []string, token string, skipWait bool, provider Provider) {
	startFleet := GetFleet(fleetName, token, provider)
	finalFleetSize := len(startFleet) + fleetCount

	if len(startFleet) > 0 {
		utils.Log.Info("Increasing fleet ", fleetName, " from size ", len(startFleet), " to ", finalFleetSize)
	}

	switch provider {
	case PROVIDER_LINODE:
		linode.SpawnFleet(fleetName, fleetCount, image, region, size, token)
	case PROVIDER_DIGITALOCEAN:
		digitalocean.SpawnFleet(fleetName, fleetCount, image, region, size, sshFingerprint, tags, token)
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
	}

	if !skipWait {
		utils.Log.Info("All spawn requests sent! Now waiting for all boxes to become ready")
		for {
			stillNotReady := false
			fleet := GetFleet(fleetName, token, provider)
			if len(fleet) == finalFleetSize {
				for i := range fleet {
					if (provider == PROVIDER_DIGITALOCEAN && fleet[i].Status != "active") || (provider == PROVIDER_LINODE && fleet[i].Status != "running") {
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
	box := GetBox(boxName, token, provider)

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
			panic(err)
		}
		defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

		// Copy stdin to the pty and the pty to stdout.
		// NOTE: The goroutine will keep reading until the next keystroke before returning.
		go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
		_, _ = io.Copy(os.Stdout, ptmx)

		return
	}
}
