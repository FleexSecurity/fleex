package vultr

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/FleexSecurity/fleex/pkg/box"
	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/vultr/govultr/v2"
	"golang.org/x/oauth2"
	//      "github.com/spf13/viper"
)

var log = logrus.New()

func GetClient(token string) *govultr.Client {
	config := &oauth2.Config{}
	ctx := context.Background()
	ts := config.TokenSource(ctx, &oauth2.Token{AccessToken: token})
	vultrClient := govultr.NewClient(oauth2.NewClient(ctx, ts))

	return vultrClient
}

// SpawnFleet spawns a Vultr fleet
func SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, token string, build bool) {
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
				spawnBox(box, image, region, size, token, build)
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
	vultrClient := GetClient(token)
	listOptions := &govultr.ListOptions{PerPage: 100}
	for {
		instances, meta, err := vultrClient.Instance.List(context.Background(), listOptions)
		if err != nil {
			log.Fatal(err)
		}

		for _, instance := range instances {
			boxes = append(boxes, box.Box{
				ID:     instance.ID,
				Label:  instance.Label,
				Status: string(instance.Status),
				IP:     instance.MainIP,
			})
		}
		if meta.Links.Next == "" {
			break
		} else {
			listOptions.Cursor = meta.Links.Next
			continue
		}
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
	return (fleet)
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

// GetImages returns a slice containing all snapshots of vultr account
func GetImages(token string) (images []box.Image) {
	vultrClient := GetClient(token)
	listOptions := &govultr.ListOptions{PerPage: 100}
	for {
		vultrImages, meta, err := vultrClient.Snapshot.List(context.Background(), listOptions)

		if err != nil {
			utils.Log.Fatal(err)
		}

		for _, image := range vultrImages {
			// Only list custom images
			images = append(images, box.Image{
				ID:      image.ID,
				Label:   image.Description,
				Created: image.DateCreated,
				Size:    image.Size,
				//Vendor:  "",
			})
		}
		if meta.Links.Next == "" {
			break
		} else {
			listOptions.Cursor = meta.Links.Next
			continue
		}
	}
	return images
}

// ListBoxes prints all active boxes of a vultr account
func ListBoxes(token string) {
	for _, instance := range GetBoxes(token) {
		fmt.Printf("%-10v %-16v %-20v %-15v\n", instance.ID, instance.Label, instance.Status, instance.IP)
	}
}

// ListImages prints snapshots of vultr account
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

func DeleteBoxByID(id string, token string) {
	vultrClient := GetClient(token)
	err := vultrClient.Instance.Delete(context.Background(), id)
	if err != nil {
		log.Fatal(err)
	}
}

func deleteBoxByLabel(label string, token string) {
	instances := GetBoxes(token)
	for _, instance := range instances {
		if instance.Label == label {
			DeleteBoxByID(instance.ID, token)
		}
	}
}

func spawnBox(name string, image string, region string, size string, token string, build bool) {
	//vultrPasswd := viper.GetString("vultr.password")
	vultrClient := GetClient(token)
	//swapSize := 512
	//booted := true
	sshKey := getSSHKey(token)
	instanceOptions := &govultr.InstanceCreateReq{}

	if build {
		os_id, err := strconv.Atoi(image)
		instanceOptions = &govultr.InstanceCreateReq{
			Region:   region,
			Plan:     size,
			Label:    name,
			OsID:     os_id,
			Hostname: name,
			SSHKeys:  []string{sshKey},
			Backups:  "disabled",
		}
		if err != nil {
			log.Fatal(err)
		}
	} else {
		instanceOptions = &govultr.InstanceCreateReq{
			Region:     region,
			Plan:       size,
			Label:      name,
			Hostname:   name,
			SnapshotID: image,
			SSHKeys:    []string{sshKey},
			Backups:    "disabled",
		}
	}
	_, err := vultrClient.Instance.Create(context.Background(), instanceOptions)

	if err != nil {
		//if strings.Contains(err.Error(), "Please try again") {
		//              continue
		//}
		fmt.Println(image)
		utils.Log.Fatal(err)
	}
}

func CreateImage(token string, instanceID string) {
	vultrClient := GetClient(token)
	snapshotOptions := &govultr.SnapshotReq{
		InstanceID:  instanceID,
		Description: "Fleex build image",
	}
	_, err := vultrClient.Snapshot.Create(context.Background(), snapshotOptions)
	if err != nil {
		log.Fatal(err)
	}
}

func getSSHKey(token string) string {
	fleex_key := sshutils.GetLocalPublicSSHKey()
	vultrClient := GetClient(token)
	keyID := KeyCheck(token, fleex_key)
	if keyID == "" {
		sshkeyOptions := &govultr.SSHKeyReq{
			Name:   "fleex_key",
			SSHKey: fleex_key,
		}
		_, err := vultrClient.SSHKey.Create(context.Background(), sshkeyOptions)
		if err != nil {
			utils.Log.Fatal(err)
		}
		keyID = KeyCheck(token, fleex_key)
	}
	return keyID
}

func KeyCheck(token string, fleex_key string) string {
	vultrClient := GetClient(token)
	listOptions := &govultr.ListOptions{PerPage: 100}
	var keyID string
	for {
		keys, meta, err := vultrClient.SSHKey.List(context.Background(), listOptions)

		if err != nil {
			utils.Log.Fatal(err)
		}
		for _, key := range keys {
			if fleex_key == key.SSHKey {
				keyID = key.ID
			} else {
				keyID = ""
			}
		}
		if meta.Links.Next == "" {
			break
		} else {
			listOptions.Cursor = meta.Links.Next
			continue
		}
	}
	return keyID
}
