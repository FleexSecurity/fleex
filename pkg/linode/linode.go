package linode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

type LinodeTemplate struct {
	BackupID        int      `json:"backup_id"`
	BackupsEnabled  bool     `json:"backups_enabled"`
	SwapSize        int      `json:"swap_size"`
	Image           string   `json:"image"`
	RootPassword    string   `json:"root_pass"`
	AuthorizedKeys  []string `json:"authorized_keys"`
	AuthorizedUsers []string `json:"authorized_users"`
	Booted          bool     `json:"booted"`
	Label           string   `json:"label"`
	LinodeType      string   `json:"type"`
	Region          string   `json:"region"`
	Group           string   `json:"group"`
}

type LinodeImage struct {
	DiskID      int    `json:"disk_id"`
	Description string `json:"description"`
	Label       string `json:"label"`
}

type LinodeDisk struct {
	Data []struct {
		ID         int    `json:"id"`
		Status     string `json:"status"`
		Label      string `json:"label"`
		Filesystem string `json:"filesystem"`
	}
}

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

	fleet := make(chan string, fleetCount)
	processGroup := new(sync.WaitGroup)
	processGroup.Add(fleetCount)

	for i := 0; i < fleetCount; i++ {
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

	fmt.Println(err)
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
		// fmt.Println(linode.ID, linode.Label, linode.Group, linode.Status, linode.IP)
		fmt.Printf("%-10v %-16v %-10v %-20v %-15v\n", linode.ID, linode.Label, linode.Group, linode.Status, linode.IP)
	}
}

// ListImages prints all private images of a Linode account
func ListImages(token string) {
	images := GetImages(token)
	for _, image := range images {
		fmt.Println(image.ID, image.Label, image.Size, image.Created, image.Vendor)
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
		utils.Log.Fatal(err)
	}
}

func CreateImage(token string, linodeID int, label string) {
	diskID := GetDiskID(token, linodeID)

	newLinode := LinodeImage{DiskID: diskID, Description: "Fleex build image", Label: label}
	postJSON, _ := json.Marshal(newLinode)

	req, err := http.NewRequest("POST", "https://api.linode.com/v4/images", bytes.NewBuffer(postJSON))
	if err != nil {
		utils.Log.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		utils.Log.Fatal(err)
	}
	defer resp.Body.Close()

	/*if resp.StatusCode == 200 {
		fmt.Println("Image created")
	}*/
}

func GetDiskID(token string, linodeID int) int {
	req, err := http.NewRequest("GET", "https://api.linode.com/v4/linode/instances/"+strconv.Itoa(linodeID)+"/disks", nil)
	if err != nil {
		utils.Log.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		utils.Log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err.Error())
	}
	var data LinodeDisk
	err = json.Unmarshal(body, &data)
	if err != nil {
		panic(err.Error())
	}

	return data.Data[0].ID
}
