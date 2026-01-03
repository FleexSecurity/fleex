package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/provider"
	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/vultr/govultr/v2"
)

type VultrService struct {
	Client  *govultr.Client
	Configs *models.Config
}

func (v VultrService) SpawnFleet(fleetName string, fleetCount int) error {
	existingFleet, _ := v.GetFleet(fleetName)
	providerName := v.Configs.Settings.Provider
	providerInfo := v.Configs.Providers[providerName]

	image := providerInfo.Image
	region := providerInfo.Region
	size := providerInfo.Size

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
				err := v.spawnBox(box, image, region, size)
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
func (v VultrService) GetBoxes() (boxes []provider.Box, err error) {
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
func (v VultrService) GetFleet(fleetName string) (fleet []provider.Box, err error) {
	boxes, err := v.GetBoxes()
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
func (v VultrService) GetBox(boxName string) (provider.Box, error) {
	// TODO manage error
	boxes, _ := v.GetBoxes()

	for _, box := range boxes {
		if box.Label == boxName {
			return box, nil
		}
	}
	return provider.Box{}, models.ErrBoxNotFound
}

func (v VultrService) GetImages() (images []provider.Image, err error) {
	listOptions := &govultr.ListOptions{PerPage: 100}
	for {
		vultrImages, meta, err := v.Client.Snapshot.List(context.Background(), listOptions)

		if err != nil {
			return []provider.Image{}, err
		}

		for _, image := range vultrImages {
			images = append(images, provider.Image{
				ID:      image.ID,
				Label:   image.Description,
				Created: image.DateCreated,
				Size:    image.Size,
			})
		}
		if meta.Links.Next == "" {
			break
		} else {
			listOptions.Cursor = meta.Links.Next
			continue
		}
	}
	return images, nil
}

func (v VultrService) ListImages() error {
	images, err := v.GetImages()
	if err != nil {
		return err
	}
	for _, image := range images {
		fmt.Printf("%-18v %-48v %-6v %-29v %-15v\n", image.ID, image.Label, image.Size, image.Created, image.Vendor)
	}
	return nil
}

func (v VultrService) RemoveImages(name string) error {
	images, err := v.GetImages()
	if err != nil {
		return err
	}
	for _, image := range images {
		if image.Label == name {
			err := v.Client.Snapshot.Delete(context.Background(), image.ID)
			if err != nil {
				return err
			}
			fmt.Println("Successfully removed:", name)
			return nil
		}
	}
	return models.ErrImageNotFound
}

func (v VultrService) DeleteFleet(name string) error {
	boxes, err := v.GetBoxes()
	if err != nil {
		return err
	}
	for _, box := range boxes {
		if box.Label == name {
			// We only have to delete a single box
			err := v.DeleteBoxByID(box.ID)
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
				err := v.DeleteBoxByID(box.ID)
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

func (v VultrService) DeleteBoxByID(id string) error {
	err := v.Client.Instance.Delete(context.Background(), id)
	if err != nil {
		return err
	}
	return nil
}

func (v VultrService) DeleteBoxByLabel(label string) error {
	boxes, err := v.GetBoxes()
	if err != nil {
		return err
	}
	for _, instance := range boxes {
		if instance.Label == label {
			err := v.DeleteBoxByID(instance.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (v VultrService) RunCommand(name, command string, port int, username, password string) error {
	boxes, err := v.GetBoxes()
	if err != nil {
		return err
	}

	privateKey := v.Configs.SSHKeys.PrivateFile

	for _, box := range boxes {
		if box.Label == name {
			sshutils.RunCommand(command, box.IP, port, username, privateKey)
			return nil
		}
	}

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
				sshutils.RunCommand(command, box.IP, port, username, privateKey)
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

func (v VultrService) spawnBox(name string, image string, region string, size string) error {
	sshKey := v.getSSHKey()
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
		return models.ErrInvalidImage
	}
	return nil
}

func (v VultrService) CreateImage(diskID int, label string) error {
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

func (v VultrService) getSSHKey() string {
	fleex_key := sshutils.GetLocalPublicSSHKey()
	keyID := v.KeyCheck(fleex_key)
	if keyID == "" {
		sshkeyOptions := &govultr.SSHKeyReq{
			Name:   "fleex_key",
			SSHKey: fleex_key,
		}
		_, err := v.Client.SSHKey.Create(context.Background(), sshkeyOptions)
		if err != nil {
			utils.Log.Fatal(err)
		}
		keyID = v.KeyCheck(fleex_key)
	}
	return keyID
}

func (v VultrService) KeyCheck(fleex_key string) string {
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
