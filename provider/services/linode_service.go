package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/FleexSecurity/fleex/provider"
	"github.com/linode/linodego"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type LinodeService struct{}

func getClient(token string) linodego.Client {
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

func (l LinodeService) SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, sshFingerprint string, tags []string, token string) {
	existingFleet := l.GetFleet(fleetName, token)

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

func (l LinodeService) GetFleet(fleetName, token string) (fleet []provider.Box) {
	boxes := l.GetBoxes(token)

	for _, box := range boxes {
		if strings.HasPrefix(box.Label, fleetName) {
			fleet = append(fleet, box)
		}
	}
	return fleet
}

func (l LinodeService) GetBox(boxName, token string) provider.Box {
	boxes := l.GetBoxes(token)

	for _, box := range boxes {
		if box.Label == boxName {
			return box
		}
	}
	utils.Log.Fatal("Box not found!")
	return provider.Box{}
}

func (l LinodeService) GetBoxes(token string) (boxes []provider.Box) {
	linodeClient := getClient(token)
	linodes, err := linodeClient.ListInstances(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	for _, linode := range linodes {
		linodeID := strconv.Itoa(linode.ID)
		boxes = append(boxes, provider.Box{
			ID:     linodeID,
			Label:  linode.Label,
			Group:  linode.Group,
			Status: string(linode.Status),
			IP:     linode.IPv4[0].String(),
		})
	}
	return boxes
}

func getImages(token string) (images []provider.Image) {
	linodeClient := getClient(token)

	linodeImages, err := linodeClient.ListImages(context.Background(), nil)

	if err != nil {
		utils.Log.Fatal(err)
	}

	for _, image := range linodeImages {
		// Only list custom images
		if strings.HasPrefix(image.ID, "private") {
			images = append(images, provider.Image{
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

func (l LinodeService) ListBoxes(token string) {
	for _, linode := range l.GetBoxes(token) {
		fmt.Printf("%-10v %-16v %-10v %-20v %-15v\n", linode.ID, linode.Label, linode.Group, linode.Status, linode.IP)
	}
}

func (l LinodeService) ListImages(token string) {
	images := getImages(token)
	for _, image := range images {
		fmt.Printf("%-18v %-48v %-6v %-29v %-15v\n", image.ID, image.Label, image.Size, image.Created, image.Vendor)
	}
}

func spawnBox(name string, image string, region string, size string, token string) {
	for {
		linPasswd := viper.GetString("linode.password")

		linodeClient := getClient(token)
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

func (l LinodeService) DeleteFleet(name string, token string) {
	boxes := l.GetBoxes(token)
	for _, box := range boxes {
		if box.Label == name {
			// We only have to delete a single box
			l.DeleteBoxByID(box.ID, token)
			return
		}
	}

	// Otherwise, we got a fleet to delete
	fleetSize := l.CountFleet(name, boxes)

	fleet := make(chan *provider.Box, fleetSize)
	processGroup := new(sync.WaitGroup)
	processGroup.Add(fleetSize)

	for i := 0; i < fleetSize; i++ {
		go func() {
			for {
				box := <-fleet

				if box == nil {
					break
				}
				l.DeleteBoxByID(box.ID, token)
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

func (l LinodeService) DeleteBoxByID(id string, token string) {
	linodeClient := getClient(token)
	linodeID, _ := strconv.Atoi(id)
	err := linodeClient.DeleteInstance(context.Background(), linodeID)
	if err != nil {
		log.Fatal(err)
	}
}

func (l LinodeService) DeleteBoxByLabel(label string, token string) {
	linodes := l.GetBoxes(token)
	for _, linode := range linodes {
		if linode.Label == label && linode.Label != "BugBountyUbuntu" {
			l.DeleteBoxByID(linode.ID, token)
		}
	}
}

func (l LinodeService) CountFleet(fleetName string, boxes []provider.Box) (count int) {
	for _, box := range boxes {
		if strings.HasPrefix(box.Label, fleetName) {
			count++
		}
	}
	return count
}

func (l LinodeService) RunCommand(name, command string, port int, username, password, token string) {
	boxes := l.GetBoxes(token)
	for _, box := range boxes {
		if box.Label == name {
			// It's a single box
			sshutils.RunCommand(command, box.IP, port, username, password)
			return
		}
	}

	// Otherwise, send command to a fleet
	fleetSize := l.CountFleet(name, boxes)

	fleet := make(chan *provider.Box, fleetSize)
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

func (l LinodeService) CreateImage(token string, diskID int, label string) {
	linodeClient := getClient(token)
	linodeID := getDiskID(token, diskID)
	_, err := linodeClient.CreateImage(context.Background(), linodego.ImageCreateOptions{
		DiskID:      linodeID,
		Description: "Fleex build image",
		Label:       label,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func getDiskID(token string, linodeID int) int {
	linodeClient := getClient(token)
	disk, err := linodeClient.ListInstanceDisks(context.Background(), linodeID, nil)
	if err != nil {
		log.Fatal(err)
	}
	return disk[0].ID
}
