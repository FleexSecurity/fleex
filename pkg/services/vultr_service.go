package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/FleexSecurity/fleex/pkg/provider"
	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/vultr/govultr/v2"
)

type VultrService struct {
	Client *govultr.Client
}

// SpawnFleet spawns a Vultr fleet
func (v VultrService) SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, sshFingerprint string, tags []string, token string) error {
	existingFleet, _ := v.GetFleet(fleetName, token)

	threads := 10
	fleet := make(chan string, threads)
	processGroup := new(sync.WaitGroup)
	processGroup.Add(threads)

	for i := 0; i < threads; i++ {
		go func() error {
			for {
				box := <-fleet

				if box == "" {
					break
				}

				utils.Log.Info("Spawning box ", box)
				err := v.spawnBox(box, image, region, size, token)
				if err != nil {
					return err
				}
			}
			processGroup.Done()
			return nil
		}()
	}

	for i := 0; i < fleetCount; i++ {
		fleet <- fleetName + "-" + strconv.Itoa(i+1+len(existingFleet))
	}

	close(fleet)
	processGroup.Wait()
	return nil
}

// GetBoxes returns a slice containg all active boxes of a Linode account
func (v VultrService) GetBoxes(token string) (boxes []provider.Box, err error) {
	listOptions := &govultr.ListOptions{PerPage: 100}
	for {
		instances, meta, err := v.Client.Instance.List(context.Background(), listOptions)
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
func (v VultrService) GetFleet(fleetName, token string) (fleet []provider.Box, err error) {
	boxes, err := v.GetBoxes(token)
	if err != nil {
		return []provider.Box{}, err
	}

	for _, box := range boxes {
		if strings.HasPrefix(box.Label, fleetName) {
			fleet = append(fleet, box)
		}
	}
	return fleet, nil
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
	listOptions := &govultr.ListOptions{PerPage: 100}
	for {
		vultrImages, meta, err := v.Client.Snapshot.List(context.Background(), listOptions)

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

// TODO
func (l VultrService) RemoveImages(token string, name string) error {
	return nil
}

func (v VultrService) DeleteFleet(name string, token string) error {
	boxes, err := v.GetBoxes(token)
	if err != nil {
		return err
	}
	for _, box := range boxes {
		if box.Label == name {
			// We only have to delete a single box
			err := v.DeleteBoxByID(box.ID, token)
			if err != nil {
				return err
			}
			return nil
		}
	}

	// Otherwise, we got a fleet to delete
	fleetSize := v.CountFleet(name, boxes)

	fleet := make(chan *provider.Box, fleetSize)
	processGroup := new(sync.WaitGroup)
	processGroup.Add(fleetSize)

	for i := 0; i < fleetSize; i++ {
		go func() error {
			for {
				box := <-fleet

				if box == nil {
					break
				}
				err := v.DeleteBoxByID(box.ID, token)
				if err != nil {
					return err
				}
			}
			processGroup.Done()
			return nil
		}()
	}

	for i := range boxes {
		if strings.HasPrefix(boxes[i].Label, name) {
			fleet <- &boxes[i]
		}
	}

	close(fleet)
	processGroup.Wait()
	return nil
}

func (v VultrService) DeleteBoxByID(id string, token string) error {
	err := v.Client.Instance.Delete(context.Background(), id)
	if err != nil {
		return err
	}
	return nil
}

func (v VultrService) DeleteBoxByLabel(label string, token string) error {
	boxes, err := v.GetBoxes(token)
	if err != nil {
		return err
	}
	for _, instance := range boxes {
		if instance.Label == label {
			err := v.DeleteBoxByID(instance.ID, token)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (v VultrService) RunCommand(name, command string, port int, username, password, token string) error {
	boxes, err := v.GetBoxes(token)
	if err != nil {
		return err
	}
	for _, box := range boxes {
		if box.Label == name {
			// It's a single box
			sshutils.RunCommand(command, box.IP, port, username, password)
			return nil
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
	return nil
}

func (v VultrService) CountFleet(fleetName string, boxes []provider.Box) (count int) {
	for _, box := range boxes {
		if strings.HasPrefix(box.Label, fleetName) {
			count++
		}
	}
	return count
}

func (v VultrService) spawnBox(name string, image string, region string, size string, token string) error {
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
			return err
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
	_, err = v.Client.Instance.Create(context.Background(), instanceOptions)

	if err != nil {
		return provider.ErrInvalidImage
	}
	return nil
}

func (v VultrService) CreateImage(token string, diskID int, label string) error {
	snapshotOptions := &govultr.SnapshotReq{
		InstanceID:  fmt.Sprint(diskID),
		Description: "Fleex build image",
	}
	_, err := v.Client.Snapshot.Create(context.Background(), snapshotOptions)
	if err != nil {
		return err
	}
	return nil
}

func (v VultrService) getSSHKey(token string) string {
	fleex_key := sshutils.GetLocalPublicSSHKey()
	keyID := v.KeyCheck(token, fleex_key)
	if keyID == "" {
		sshkeyOptions := &govultr.SSHKeyReq{
			Name:   "fleex_key",
			SSHKey: fleex_key,
		}
		_, err := v.Client.SSHKey.Create(context.Background(), sshkeyOptions)
		if err != nil {
			utils.Log.Fatal(err)
		}
		keyID = v.KeyCheck(token, fleex_key)
	}
	return keyID
}

func (v VultrService) KeyCheck(token string, fleex_key string) string {
	listOptions := &govultr.ListOptions{PerPage: 100}
	var keyID string
	for {
		keys, meta, err := v.Client.SSHKey.List(context.Background(), listOptions)

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
