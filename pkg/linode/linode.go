package linode

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/linode/linodego"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/box"
	"github.com/sw33tLie/fleex/pkg/utils"
	"golang.org/x/oauth2"

	"github.com/sw33tLie/fleex/pkg/sshutils"
)

var log = logrus.New()

func GetClient(token string) linodego.Client {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	linodeClient := linodego.NewClient(oauth2Client)
	linodeClient.SetDebug(false)

	return linodeClient
}

// SpawnFleet spawns a Linode fleet
func SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, token string) {
	existingFleet := GetFleet(fleetName, token)

	threads := 10
	fleet := make(chan string, threads)
	processGroup := new(sync.WaitGroup)
	processGroup.Add(threads)

	for i := 0; i < threads; i++ {
		go func() {
			for {
				box := <-fleet

				if box == "" {
					break
				}

				utils.Log.Info("Spawning box ", box)
				spawnBox(box, image, region, size, token)
			}
			processGroup.Done()
		}()
	}

	for i := 0; i < fleetCount; i++ {
		fleet <- fleetName + "-" + strconv.Itoa(i+1+len(existingFleet))
	}

	close(fleet)
	processGroup.Wait()
}

// GetBoxes returns a slice containg all active boxes of a Linode account
func GetBoxes(token string) (boxes []box.Box) {
	linodeClient := GetClient(token)
	linodes, err := linodeClient.ListInstances(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	for _, linode := range linodes {
		boxes = append(boxes, box.Box{
			ID:     linode.ID,
			Label:  linode.Label,
			Group:  linode.Group,
			Status: string(linode.Status),
			IP:     linode.IPv4[0].String(),
		})
	}
	return boxes
}

// GetBoxes returns a slice containg all boxes of a given fleet
func GetFleet(fleetName, token string) (fleet []box.Box) {
	boxes := GetBoxes(token)

	for _, box := range boxes {
		if strings.HasPrefix(box.Label, fleetName) {
			fleet = append(fleet, box)
		}
	}
	return fleet
}

// GetBox returns a single box by its label
func GetBox(boxName, token string) box.Box {
	boxes := GetBoxes(token)

	for _, box := range boxes {
		if box.Label == boxName {
			return box
		}
	}
	utils.Log.Fatal("Box not found!")
	return box.Box{}
}

// GetImages returns a slice containing all private images of a Linode account
func GetImages(token string) (images []box.Image) {
	linodeClient := GetClient(token)

	linodeImages, err := linodeClient.ListImages(context.Background(), nil)

	if err != nil {
		utils.Log.Fatal(err)
	}

	for _, image := range linodeImages {
		// Only list custom images
		if strings.HasPrefix(image.ID, "private") {
			images = append(images, box.Image{
				ID:      image.ID,
				Label:   image.Label,
				Created: image.Created.String(),
				Size:    image.Size,
				Vendor:  image.Vendor,
			})
		}
	}
	return images
}

// ListBoxes prints all active boxes of a Linode account
func ListBoxes(token string) {
	for _, linode := range GetBoxes(token) {
		fmt.Printf("%-10v %-16v %-10v %-20v %-15v\n", linode.ID, linode.Label, linode.Group, linode.Status, linode.IP)
	}
}

// ListImages prints all private images of a Linode account
func ListImages(token string) {
	images := GetImages(token)
	for _, image := range images {
		fmt.Printf("%-18v %-48v %-6v %-29v %-15v\n", image.ID, image.Label, image.Size, image.Created, image.Vendor)
	}
}

func DeleteFleet(name string, token string) {
	boxes := GetBoxes(token)
	for _, box := range boxes {
		if box.Label == name {
			// We only have to delete a single box
			DeleteBoxByID(box.ID, token)
			return
		}
	}

	// Otherwise, we got a fleet to delete
	fleetSize := CountFleet(name, boxes)

	fleet := make(chan *box.Box, fleetSize)
	processGroup := new(sync.WaitGroup)
	processGroup.Add(fleetSize)

	for i := 0; i < fleetSize; i++ {
		go func() {
			for {
				box := <-fleet

				if box == nil {
					break
				}
				DeleteBoxByID(box.ID, token)
			}
			processGroup.Done()
		}()
	}

	for i := range boxes {
		if strings.HasPrefix(boxes[i].Label, name) {
			fleet <- &boxes[i]
		}
	}

	close(fleet)
	processGroup.Wait()
}

func RunCommand(name, command string, port int, username, password, token string) {
	boxes := GetBoxes(token)
	for _, box := range boxes {
		if box.Label == name {
			// It's a single box
			sshutils.RunCommand(command, box.IP, port, username, password)
			return
		}
	}

	// Otherwise, send command to a fleet
	fleetSize := CountFleet(name, boxes)

	fleet := make(chan *box.Box, fleetSize)
	processGroup := new(sync.WaitGroup)
	processGroup.Add(fleetSize)

	for i := 0; i < fleetSize; i++ {
		go func() {
			for {
				box := <-fleet

				if box == nil {
					break
				}
				sshutils.RunCommand(command, box.IP, port, username, password)
			}
			processGroup.Done()
		}()
	}

	for i := range boxes {
		if strings.HasPrefix(boxes[i].Label, name) {
			fleet <- &boxes[i]
		}
	}

	close(fleet)
	processGroup.Wait()
}

func CountFleet(fleetName string, boxes []box.Box) (count int) {
	for _, box := range boxes {
		if strings.HasPrefix(box.Label, fleetName) {
			count++
		}
	}
	return count
}

func DeleteBoxByID(id int, token string) {
	linodeClient := GetClient(token)
	err := linodeClient.DeleteInstance(context.Background(), id)
	if err != nil {
		log.Fatal(err)
	}
}

func deleteBoxByLabel(label string, token string) {
	linodes := GetBoxes(token)
	for _, linode := range linodes {
		if linode.Label == label && linode.Label != "BugBountyUbuntu" {
			DeleteBoxByID(linode.ID, token)
		}
	}
}

func spawnBox(name string, image string, region string, size string, token string) {
	for {
		linPasswd := viper.GetString("linode.password")

		linodeClient := GetClient(token)
		swapSize := 512
		booted := true
		_, err := linodeClient.CreateInstance(context.Background(), linodego.InstanceCreateOptions{
			SwapSize:       &swapSize,
			Image:          image,
			RootPass:       linPasswd,
			Type:           size,
			Region:         region,
			AuthorizedKeys: []string{sshutils.GetLocalPublicSSHKey()},
			Booted:         &booted,
			Label:          name,
		})

		if err != nil {
			if strings.Contains(err.Error(), "Please try again") {
				continue
			}
			utils.Log.Fatal(err)
		}
		break
	}
}

func CreateImage(token string, linodeID int, label string) {
	linodeClient := GetClient(token)
	diskID := GetDiskID(token, linodeID)
	_, err := linodeClient.CreateImage(context.Background(), linodego.ImageCreateOptions{
		DiskID:      diskID,
		Description: "Fleex build image",
		Label:       label,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func GetDiskID(token string, linodeID int) int {
	linodeClient := GetClient(token)
	disk, err := linodeClient.ListInstanceDisks(context.Background(), linodeID, nil)
	if err != nil {
		log.Fatal(err)
	}
	return disk[0].ID
}
