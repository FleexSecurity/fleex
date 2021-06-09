package linode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sw33tLie/fleex/pkg/sshutils"
	"github.com/tidwall/gjson"
)

type LinodeBox struct {
	ID     int
	Label  string
	Group  string
	Status string
	IP     string
}

type LinodeImage struct {
	ID      string
	Label   string
	Created string
	Size    int
	Vendor  string
}

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

// SpawnFleet spawns a Linode fleet
func SpawnFleet(fleetName string, fleetCount int, image string, region string, token string) {
	fleetNames := make(chan string, fleetCount)
	processGroup := new(sync.WaitGroup)
	processGroup.Add(fleetCount)

	for i := 0; i < fleetCount; i++ {
		go func() {
			for {

				linodeName := <-fleetNames

				if linodeName == "" {
					break
				}

				fmt.Println("SPAWNING " + linodeName)
				spawnBox(linodeName, image, region, token)

			}
			processGroup.Done()
		}()
	}

	for i := 0; i < fleetCount; i++ {
		fleetNames <- fleetName + "-" + strconv.Itoa(i+1)
	}

	close(fleetNames)
	processGroup.Wait()
}

func GetBoxes(token string) (boxes []LinodeBox) {
	// API Docs: https://developers.linode.com/api/v4/linode-instances
	req, err := http.NewRequest("GET", "https://api.linode.com/v4/linode/instances", nil)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal("Error. HTTP status code: " + resp.Status)
		return nil
	}

	data := gjson.GetMany(string(body), "data.#.id", "data.#.label", "data.#.group", "data.#.status", "data.#.ipv4")

	for i := 0; i < len(data[0].Array()); i++ {
		boxes = append(boxes, LinodeBox{int(data[0].Array()[i].Int()), data[1].Array()[i].Str, data[2].Array()[i].Str, data[3].Array()[i].Str, data[4].Array()[i].Array()[0].Str})
	}
	return boxes
}

func GetImages(token string) (images []LinodeImage) {
	req, err := http.NewRequest("GET", "https://api.linode.com/v4/images", nil)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal("Error. HTTP status code: " + resp.Status)
		return nil
	}

	data := gjson.GetMany(string(body), "data.#.id", "data.#.label", "data.#.created", "data.#.size", "data.#.vendor")

	for i := 0; i < len(data[0].Array()); i++ {
		// Only list custom images
		if strings.HasPrefix(data[0].Array()[i].Str, "private") {
			images = append(images, LinodeImage{data[0].Array()[i].Str, data[1].Array()[i].Str, data[2].Array()[i].Str, int(data[3].Array()[i].Int()), data[4].Array()[i].Str})
		}
	}
	return images
}

func ListBoxes(token string) {
	linodes := GetBoxes(token)
	for _, linode := range linodes {
		fmt.Println(linode.ID, linode.Label, linode.Group, linode.Status, linode.IP)
	}
}

func ListImages(token string) {
	images := GetImages(token)
	for _, image := range images {
		fmt.Println(image.ID, image.Label, image.Size, image.Created, image.Vendor)
	}
}

func DeleteFleetOrBox(name string, token string) {
	linodes := GetBoxes(token)
	for _, linode := range linodes {
		if linode.Label == name {
			// It's a single box
			deleteBoxByID(linode.ID, token)
			return
		}
	}

	// Otherwise, we got a fleet to delete

}

func RunCommand(name string, command string, token string) {
	linodes := GetBoxes(token)
	for _, linode := range linodes {
		if linode.Label == name {
			sshutils.RunCommand(command, linode.IP, 2266, "op", "1337superPass")
			return
		}
	}
}

func deleteBoxByID(id int, token string) {
	for {
		req, err := http.NewRequest("DELETE", "https://api.linode.com/v4/linode/instances/"+strconv.Itoa(id), nil)
		if err != nil {
			log.Fatal(err)
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
			deleteBoxByID(linode.ID, token)
		}
	}
}

func spawnBox(name string, image string, region string, token string) {
	for {
		newLinode := LinodeTemplate{SwapSize: 512, Image: image, RootPassword: "1337superPass", LinodeType: "g6-nanode-1", Region: region, AuthorizedKeys: []string{sshutils.GetLocalPublicSSHKey()}, Booted: true, Label: name}
		postJSON, err := json.Marshal(newLinode)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(bytes.NewBuffer(postJSON))
		req, err := http.NewRequest("POST", "https://api.linode.com/v4/linode/instances", bytes.NewBuffer(postJSON))
		if err != nil {
			log.Fatal(err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(body))
		if !strings.Contains(string(body), "Please try again") {
			break
		}
		time.Sleep(5 * time.Second)
	}
}
