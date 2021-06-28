package linode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/box"
	"github.com/sw33tLie/fleex/pkg/utils"

	"github.com/sw33tLie/fleex/pkg/sshutils"
	"github.com/tidwall/gjson"
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

// SpawnFleet spawns a Linode fleet
func SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, token string, wait bool) {
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

	if fleetCount > 1 {
		for i := 0; i < fleetCount; i++ {
			fleet <- fleetName + "-" + strconv.Itoa(i+1+len(existingFleet))
		}
	} else {
		fleet <- fleetName
	}

	close(fleet)
	processGroup.Wait()

	if wait {
		for {
			stillNotReady := false
			fleet := GetFleet(fleetName, token)
			if len(fleet) == fleetCount {
				for i := range fleet {
					if fleet[i].Status != "running" {
						stillNotReady = true
					}
				}
			}

			if stillNotReady {
				time.Sleep(8 * time.Second)
			} else {
				break
			}
		}
	}
}

// GetBoxes returns a slice containg all active boxes of a Linode account
func GetBoxes(token string) (boxes []box.Box) {
	req, err := http.NewRequest("GET", "https://api.linode.com/v4/linode/instances", nil)
	if err != nil {
		utils.Log.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		utils.Log.Fatal("Error. HTTP status code: " + resp.Status)
		return nil
	}

	data := gjson.GetMany(string(body), "data.#.id", "data.#.label", "data.#.group", "data.#.status", "data.#.ipv4")

	for i := 0; i < len(data[0].Array()); i++ {
		boxes = append(boxes, box.Box{int(data[0].Array()[i].Int()), data[1].Array()[i].Str, data[2].Array()[i].Str, data[3].Array()[i].Str, data[4].Array()[i].Array()[0].Str})
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
	req, err := http.NewRequest("GET", "https://api.linode.com/v4/images", nil)
	if err != nil {
		utils.Log.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		utils.Log.Fatal("Error. HTTP status code: " + resp.Status)
		return nil
	}

	data := gjson.GetMany(string(body), "data.#.id", "data.#.label", "data.#.created", "data.#.size", "data.#.vendor")

	for i := 0; i < len(data[0].Array()); i++ {
		// Only list custom images
		if strings.HasPrefix(data[0].Array()[i].Str, "private") {
			images = append(images, box.Image{data[0].Array()[i].Str, data[1].Array()[i].Str, data[2].Array()[i].Str, int(data[3].Array()[i].Int()), data[4].Array()[i].Str})
		}
	}
	return images
}

// ListBoxes prints all active boxes of a Linode account
func ListBoxes(token string) {
	linodes := GetBoxes(token)
	for _, linode := range linodes {
		fmt.Println(linode.ID, linode.Label, linode.Group, linode.Status, linode.IP)
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
	for {
		req, err := http.NewRequest("DELETE", "https://api.linode.com/v4/linode/instances/"+strconv.Itoa(id), nil)
		if err != nil {
			utils.Log.Fatal(err)
		}

		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			break
		}

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
	for {
		newLinode := LinodeTemplate{SwapSize: 512, Image: image, RootPassword: linPasswd, LinodeType: size, Region: region, AuthorizedKeys: []string{sshutils.GetLocalPublicSSHKey()}, Booted: true, Label: name}
		postJSON, err := json.Marshal(newLinode)
		if err != nil {
			utils.Log.Fatal(err)
		}

		req, err := http.NewRequest("POST", "https://api.linode.com/v4/linode/instances", bytes.NewBuffer(postJSON))
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

		body, _ := ioutil.ReadAll(resp.Body)
		utils.Log.Debug("API Response: ", string(body))
		if !strings.Contains(string(body), "Please try again") {
			break
		}
		time.Sleep(5 * time.Second)
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
