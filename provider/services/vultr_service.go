package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/FleexSecurity/fleex/provider"
	"github.com/vultr/govultr/v2"
	"golang.org/x/oauth2"
)

type VultrService struct{}

func (v VultrService) getClient(token string) *govultr.Client {
	config := &oauth2.Config{}
	ctx := context.Background()
	ts := config.TokenSource(ctx, &oauth2.Token{AccessToken: token})
	vultrClient := govultr.NewClient(oauth2.NewClient(ctx, ts))

	return vultrClient
}

// SpawnFleet spawns a Vultr fleet
// func (l LinodeService) SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, sshFingerprint string, tags []string, token string) {
func (v VultrService) SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, sshFingerprint string, tags []string, token string) {
	existingFleet := v.GetFleet(fleetName, token)

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
				v.spawnBox(box, image, region, size, token)
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
func (v VultrService) GetBoxes(token string) (boxes []provider.Box, err error) {
	vultrClient := v.getClient(token)
	listOptions := &govultr.ListOptions{PerPage: 100}
	for {
		instances, meta, err := vultrClient.Instance.List(context.Background(), listOptions)
		if err != nil {
			log.Fatal(err)
		}

		for _, instance := range instances {
			boxes = append(boxes, provider.Box{
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
	return boxes, nil
}

// GetBoxes returns a slice containg all boxes of a given fleet
func (v VultrService) GetFleet(fleetName, token string) (fleet []provider.Box) {
	// TODO manage error
	boxes, _ := v.GetBoxes(token)

	for _, box := range boxes {
		if strings.HasPrefix(box.Label, fleetName) {
			fleet = append(fleet, box)
		}
	}
	return (fleet)
}

// GetBox returns a single box by its label
func (v VultrService) GetBox(boxName, token string) (provider.Box, error) {
	// TODO manage error
	boxes, _ := v.GetBoxes(token)

	for _, box := range boxes {
		if box.Label == boxName {
			return box, nil
		}
	}
	return provider.Box{}, provider.ErrBoxNotFound
}

// GetImages returns a slice containing all snapshots of vultr account
func (v VultrService) GetImages(token string) (images []provider.Image) {
	vultrClient := v.getClient(token)
	listOptions := &govultr.ListOptions{PerPage: 100}
	for {
		vultrImages, meta, err := vultrClient.Snapshot.List(context.Background(), listOptions)

		if err != nil {
			utils.Log.Fatal(err)
		}

		for _, image := range vultrImages {
			// Only list custom images
			images = append(images, provider.Image{
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
func (v VultrService) ListBoxes(token string) {
	// TODO manage error
	boxes, _ := v.GetBoxes(token)
	for _, instance := range boxes {
		fmt.Printf("%-10v %-16v %-20v %-15v\n", instance.ID, instance.Label, instance.Status, instance.IP)
	}
}

// ListImages prints snapshots of vultr account
func (v VultrService) ListImages(token string) error {
	images := v.GetImages(token)
	for _, image := range images {
		fmt.Printf("%-18v %-48v %-6v %-29v %-15v\n", image.ID, image.Label, image.Size, image.Created, image.Vendor)
	}
	return nil
}

func (v VultrService) DeleteFleet(name string, token string) {
	// TODO manage error
	boxes, _ := v.GetBoxes(token)
	for _, box := range boxes {
		if box.Label == name {
			// We only have to delete a single box
			v.DeleteBoxByID(box.ID, token)
			return
		}
	}

	// Otherwise, we got a fleet to delete
	fleetSize := v.CountFleet(name, boxes)

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
				v.DeleteBoxByID(box.ID, token)
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

func (v VultrService) RunCommand(name, command string, port int, username, password, token string) {
	// TODO manage error
	boxes, _ := v.GetBoxes(token)
	for _, box := range boxes {
		if box.Label == name {
			// It's a single box
			sshutils.RunCommand(command, box.IP, port, username, password)
			return
		}
	}

	// Otherwise, send command to a fleet
	fleetSize := v.CountFleet(name, boxes)

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

func (v VultrService) CountFleet(fleetName string, boxes []provider.Box) (count int) {
	for _, box := range boxes {
		if strings.HasPrefix(box.Label, fleetName) {
			count++
		}
	}
	return count
}

func (v VultrService) DeleteBoxByID(id string, token string) {
	vultrClient := v.getClient(token)
	err := vultrClient.Instance.Delete(context.Background(), id)
	if err != nil {
		log.Fatal(err)
	}
}

func (v VultrService) DeleteBoxByLabel(label string, token string) {
	// TODO manage error
	instances, _ := v.GetBoxes(token)
	for _, instance := range instances {
		if instance.Label == label {
			v.DeleteBoxByID(instance.ID, token)
		}
	}
}

func (v VultrService) spawnBox(name string, image string, region string, size string, token string) {
	//vultrPasswd := viper.GetString("vultr.password")
	vultrClient := v.getClient(token)
	//swapSize := 512
	//booted := true
	sshKey := v.getSSHKey(token)
	instanceOptions := &govultr.InstanceCreateReq{}

	os_id, err := strconv.Atoi(image)
	if err == nil {
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
	_, err = vultrClient.Instance.Create(context.Background(), instanceOptions)

	if err != nil {
		utils.Log.Fatal(provider.ErrInvalidImage)
	}
}

func (v VultrService) CreateImage(token string, diskID int, label string) error {
	vultrClient := v.getClient(token)
	snapshotOptions := &govultr.SnapshotReq{
		InstanceID:  fmt.Sprint(diskID),
		Description: "Fleex build image",
	}
	_, err := vultrClient.Snapshot.Create(context.Background(), snapshotOptions)
	if err != nil {
		return err
	}
	return nil
}

func (v VultrService) getSSHKey(token string) string {
	fleex_key := sshutils.GetLocalPublicSSHKey()
	vultrClient := v.getClient(token)
	keyID := v.KeyCheck(token, fleex_key)
	if keyID == "" {
		sshkeyOptions := &govultr.SSHKeyReq{
			Name:   "fleex_key",
			SSHKey: fleex_key,
		}
		_, err := vultrClient.SSHKey.Create(context.Background(), sshkeyOptions)
		if err != nil {
			utils.Log.Fatal(err)
		}
		keyID = v.KeyCheck(token, fleex_key)
	}
	return keyID
}

func (v VultrService) KeyCheck(token string, fleex_key string) string {
	vultrClient := v.getClient(token)
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
