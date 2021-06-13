package linode

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hnakamur/go-scp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/box"

	"github.com/sw33tLie/fleex/pkg/sshutils"
	"github.com/sw33tLie/fleex/pkg/utils"
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

var log = logrus.New()

// SpawnFleet spawns a Linode fleet
func SpawnFleet(fleetName string, fleetCount int, image string, region string, token string, wait bool) {
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

				log.Info("Spawning box ", box)
				spawnBox(box, image, region, token)
			}
			processGroup.Done()
		}()
	}

	for i := 0; i < fleetCount; i++ {
		fleet <- fleetName + "-" + strconv.Itoa(i+1)
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

// GetImages returns a slice containing all private images of a Linode account
func GetImages(token string) (images []box.Image) {
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

func RunCommand(name string, command string, token string) {
	boxes := GetBoxes(token)
	for _, box := range boxes {
		if box.Label == name {
			// It's a single box
			sshutils.RunCommand(command, box.IP, 2266, "op", "1337superPass")
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
				sshutils.RunCommand(command, box.IP, 2266, "op", "1337superPass")
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

// TODO Polish this code
func Scan(fleetName string, command string, delete bool, input string, output string, token string) {
	fmt.Println("Scan started. Input: ", input, " output: ", output)

	// Make local temp folder
	tempFolder := path.Join("/tmp", strconv.FormatInt(time.Now().UnixNano(), 10))

	// Create temp folder
	err := os.Mkdir(tempFolder, 0755)
	if err != nil {
		log.Fatal(err)
	}

	// Input file to string
	inputString := utils.FileToString(input)

	fleet := GetFleet(fleetName, token)

	if len(fleet) < 1 {
		log.Fatal("No fleet found")
	}

	linesCount := utils.LinesCount(inputString)
	linesPerChunk := linesCount / len(fleet)

	// Iterate over multiline input string
	scanner := bufio.NewScanner(strings.NewReader(inputString))

	counter := 1
	chunkContent := ""
	var inputFiles []string
	for scanner.Scan() {
		line := scanner.Text()
		chunkContent += line + "\n"
		if counter%linesPerChunk == 0 {
			// Remove bottom empty line
			chunkContent = strings.TrimSuffix(chunkContent, "\n")
			// Save chunk
			chunkPath := path.Join(tempFolder, "chunk-"+fleetName+"-"+strconv.Itoa(counter/linesPerChunk))
			utils.StringToFile(chunkPath, chunkContent)
			inputFiles = append(inputFiles, chunkPath)

			fmt.Println("" + chunkPath)

			chunkContent = ""
		}
		counter++
	}

	if scanner.Err() != nil {
		log.Println(scanner.Err())
	}

	// Send SSH commands to all boxes

	fleetNames := make(chan *box.Box, len(fleet))
	processGroup := new(sync.WaitGroup)
	processGroup.Add(len(fleet))

	for i := 0; i < len(fleet); i++ {
		go func() {
			for {
				l := <-fleetNames

				if l == nil {
					break
				}

				linodeName := l.Label

				// Send input file via SCP
				err := scp.NewSCP(sshutils.GetConnection(l.IP, 2266, "op", "1337superPass").Client).SendFile(path.Join(tempFolder, "chunk-"+linodeName), "/home/op")
				if err != nil {
					log.Fatalf("Failed to send file: %s", err)
				}

				// Replace labels and craft final command
				finalCommand := command
				finalCommand = strings.ReplaceAll(finalCommand, "{{INPUT}}", path.Join("/home/op", "chunk-"+linodeName))
				finalCommand = strings.ReplaceAll(finalCommand, "{{OUTPUT}}", "chunk-res-"+linodeName)

				fmt.Println("SCANNING WITH ", path.Join(tempFolder, "chunk-"+linodeName), " ")
				// TODO: Not optimal, it runs GetBoxes() every time which is dumb, should use a function that does the same but by id
				RunCommand(linodeName, finalCommand, token)

				// Now download the output file
				err = scp.NewSCP(sshutils.GetConnection(l.IP, 2266, "op", "1337superPass").Client).ReceiveFile("chunk-res-"+linodeName, path.Join(tempFolder, "chunk-res-"+linodeName))
				if err != nil {
					log.Fatalf("Failed to get file: %s", err)
				}

				if delete {
					// TODO: Not the best way to delete a box, if this program crashes/is stopped
					// before reaching this line the box won't be deleted. It's better to setup
					// a cron/command on the box directly.
					DeleteBoxByID(l.ID, token)
				}

			}
			processGroup.Done()
		}()
	}

	for i := range fleet {
		fleetNames <- &fleet[i]
	}

	close(fleetNames)
	processGroup.Wait()

	// Scan done, process results

	fmt.Println("SCAN DONE")
}

func DeleteBoxByID(id int, token string) {
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
			DeleteBoxByID(linode.ID, token)
		}
	}
}

func spawnBox(name string, image string, region string, token string) {
	linPasswd := viper.GetString("linode.password")
	for {
		newLinode := LinodeTemplate{SwapSize: 512, Image: image, RootPassword: linPasswd, LinodeType: "g6-nanode-1", Region: region, AuthorizedKeys: []string{sshutils.GetLocalPublicSSHKey()}, Booted: true, Label: name}
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
