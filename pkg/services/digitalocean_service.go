package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/provider"
	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/digitalocean/godo"
)

type DigitaloceanService struct {
	Client  *godo.Client
	Configs *models.Config
}

func (d DigitaloceanService) ensureSSHKey() (string, error) {
	ctx := context.TODO()
	publicKey := sshutils.GetLocalPublicSSHKey()
	fingerprint := sshutils.SSHFingerprintGen(d.Configs.SSHKeys.PublicFile)

	opt := &godo.ListOptions{Page: 1, PerPage: 200}
	keys, _, err := d.Client.Keys.List(ctx, opt)
	if err != nil {
		return "", err
	}

	for _, key := range keys {
		if key.Fingerprint == fingerprint {
			return fingerprint, nil
		}
	}

	createRequest := &godo.KeyCreateRequest{
		Name:      "fleex",
		PublicKey: publicKey,
	}
	newKey, _, err := d.Client.Keys.Create(ctx, createRequest)
	if err != nil {
		return "", err
	}

	return newKey.Fingerprint, nil
}

func (d DigitaloceanService) SpawnFleet(fleetName string, fleetCount int) error {
	existingFleet, _ := d.GetFleet(fleetName)
	providerName := d.Configs.Settings.Provider
	providerInfo := d.Configs.Providers[providerName]

	ctx := context.TODO()
	password := providerInfo.Password
	if password == "" {
		password = "1337rootPass"
	}
	image := providerInfo.Image
	region := providerInfo.Region
	size := providerInfo.Size
	tags := providerInfo.Tags

	sshFingerprint, err := d.ensureSSHKey()
	if err != nil {
		return fmt.Errorf("failed to ensure SSH key: %w", err)
	}

	droplets := []string{}

	for i := 0; i < fleetCount; i++ {
		droplets = append(droplets, fleetName+"-"+strconv.Itoa(i+1+len(existingFleet)))
	}

	user_data := `#!/bin/bash
sudo sed -i "/^[^#]*PasswordAuthentication[[:space:]]no/c\PasswordAuthentication yes" /etc/ssh/sshd_config
sudo service sshd restart
echo 'op:` + password + `' | sudo chpasswd`

	// DigitalOcean limits CreateMultiple to 10 droplets per request
	const batchSize = 10
	imageIntID, _ := strconv.Atoi(image)
	isImageID := imageIntID > 0

	for i := 0; i < len(droplets); i += batchSize {
		end := i + batchSize
		if end > len(droplets) {
			end = len(droplets)
		}
		batch := droplets[i:end]
		batchNum := (i / batchSize) + 1
		totalBatches := (len(droplets) + batchSize - 1) / batchSize

		var createRequest *godo.DropletMultiCreateRequest
		if isImageID {
			createRequest = &godo.DropletMultiCreateRequest{
				Names:    batch,
				Region:   region,
				Size:     size,
				UserData: user_data,
				Image: godo.DropletCreateImage{
					ID: imageIntID,
				},
				SSHKeys: []godo.DropletCreateSSHKey{
					{Fingerprint: sshFingerprint},
				},
				Tags: tags,
			}
		} else {
			createRequest = &godo.DropletMultiCreateRequest{
				Names:    batch,
				Region:   region,
				Size:     size,
				UserData: user_data,
				Image: godo.DropletCreateImage{
					Slug: image,
				},
				SSHKeys: []godo.DropletCreateSSHKey{
					{Fingerprint: sshFingerprint},
				},
				Tags: tags,
			}
		}

		_, _, err := d.Client.Droplets.CreateMultiple(ctx, createRequest)
		if err != nil {
			return fmt.Errorf("batch %d/%d failed: %w", batchNum, totalBatches, err)
		}

		// Small delay between batches to avoid overwhelming the API
		if end < len(droplets) {
			time.Sleep(200 * time.Millisecond)
		}
	}

	return nil
}

func (d DigitaloceanService) GetFleet(fleetName string) (fleet []provider.Box, err error) {
	boxes, err := d.GetBoxes()
	if err != nil {
		return []provider.Box{}, err
	}

	for _, box := range boxes {
		if utils.MatchesFleetName(box.Label, fleetName) {
			fleet = append(fleet, box)
		}
	}
	return fleet, nil
}

// GetBox returns a single box by its label
func (d DigitaloceanService) GetBox(boxName string) (provider.Box, error) {
	// TODO manage error
	boxes, _ := d.GetBoxes()

	for _, box := range boxes {
		if box.Label == boxName {
			return box, nil
		}
	}
	return provider.Box{}, models.ErrBoxNotFound
}

func (d DigitaloceanService) GetBoxes() (boxes []provider.Box, err error) {
	ctx := context.TODO()

	// DigitalOcean API has a max of 200 per page, so we need to paginate
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	for {
		droplets, resp, err := d.Client.Droplets.List(ctx, opt)
		if err != nil {
			return []provider.Box{}, err
		}

		for _, droplet := range droplets {
			ip, _ := droplet.PublicIPv4()
			dID := strconv.Itoa(droplet.ID)
			boxes = append(boxes, provider.Box{ID: dID, Label: droplet.Name, Group: "", Status: droplet.Status, IP: ip})
		}

		// Check if there are more pages
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		opt.Page++
	}

	return boxes, nil
}

func (d DigitaloceanService) GetImages() (images []provider.Image, err error) {
	ctx := context.TODO()
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 9999,
	}

	doImages, _, err := d.Client.Images.ListUser(ctx, opt)
	if err != nil {
		return []provider.Image{}, err
	}

	for _, image := range doImages {
		images = append(images, provider.Image{
			ID:      strconv.Itoa(image.ID),
			Label:   image.Name,
			Size:    int(image.SizeGigaBytes),
			Status:  image.Status,
			Regions: image.Regions,
		})
	}
	return images, nil
}

func (d DigitaloceanService) ListImages() error {
	images, err := d.GetImages()
	if err != nil {
		return err
	}

	fmt.Printf("%-12s  %-40s  %-6s  %-10s  %s\n", "ID", "NAME", "SIZE", "STATUS", "REGIONS")
	fmt.Println(strings.Repeat("-", 100))
	for _, image := range images {
		regions := strings.Join(image.Regions, ",")
		fmt.Printf("%-12s  %-40s  %-4dGB  %-10s  %s\n", image.ID, image.Label, image.Size, image.Status, regions)
	}
	return nil
}

func (d DigitaloceanService) GetImageRegions(imageID int) ([]string, error) {
	ctx := context.TODO()
	image, _, err := d.Client.Images.GetByID(ctx, imageID)
	if err != nil {
		return nil, err
	}
	return image.Regions, nil
}

func (d DigitaloceanService) RemoveImages(name string) error {
	ctx := context.TODO()
	images, err := d.GetImages()
	if err != nil {
		return err
	}
	for _, image := range images {
		if image.Label == name {
			imageID, _ := strconv.Atoi(image.ID)
			_, err := d.Client.Images.Delete(ctx, imageID)
			if err != nil {
				return err
			}
			fmt.Println("Successfully removed:", name)
			return nil
		}
	}
	return models.ErrImageNotFound
}

func (d DigitaloceanService) DeleteFleet(name string) error {
	boxes, err := d.GetBoxes()
	if err != nil {
		return err
	}
	for _, droplet := range boxes {
		if droplet.Label == name {
			// It's a single box
			err := d.DeleteBoxByID(droplet.ID)
			if err != nil {
				return err
			}
			return nil
		}
	}

	// Otherwise, we got a fleet to delete
	// Continue deleting even if some fail (e.g., rate limits, transient errors)
	var lastErr error
	for _, droplet := range boxes {
		if utils.MatchesFleetName(droplet.Label, name) {
			err := d.DeleteBoxByID(droplet.ID)
			if err != nil {
				return err
			}
		}
	}
	return lastErr
}

func (d DigitaloceanService) DeleteBoxByID(ID string) error {
	ctx := context.TODO()

	ID1, err := strconv.Atoi(ID)
	if err != nil {
		return err
	}
	_, err = d.Client.Droplets.Delete(ctx, ID1)
	if err != nil {
		return err
	}
	return nil
}

func (l DigitaloceanService) DeleteBoxByLabel(label string) error {
	boxes, err := l.GetBoxes()
	if err != nil {
		return err
	}
	for _, box := range boxes {
		if box.Label == label && box.Label != "BugBountyUbuntu" {
			err := l.DeleteBoxByID(box.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d DigitaloceanService) CountFleet(fleetName string, boxes []provider.Box) (count int) {
	for _, box := range boxes {
		if utils.MatchesFleetName(box.Label, fleetName) {
			count++
		}
	}
	return count
}

func (d DigitaloceanService) RunCommand(name, command string, port int, username, password string) error {
	boxes, err := d.GetBoxes()
	if err != nil {
		return err
	}

	privateKey := d.Configs.SSHKeys.PrivateFile

	for _, box := range boxes {
		if box.Label == name {
			sshutils.RunCommand(command, box.IP, port, username, privateKey)
			return nil
		}
	}

	fleetSize := d.CountFleet(name, boxes)

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
		if utils.MatchesFleetName(boxes[i].Label, name) {
			fleet <- &boxes[i]
		}
	}

	close(fleet)
	processGroup.Wait()
	return nil
}

func (d DigitaloceanService) CreateImage(diskID int, label string) error {
	ctx := context.TODO()

	_, _, err := d.Client.DropletActions.Snapshot(ctx, diskID, label)
	if err != nil {
		return err
	}
	return nil
}

func (d DigitaloceanService) TransferImage(imageID int, region string) error {
	ctx := context.TODO()

	action, _, err := d.Client.ImageActions.Transfer(ctx, imageID, &godo.ActionRequest{
		"type":   "transfer",
		"region": region,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Transfer initiated (action ID: %d). This may take several minutes.\n", action.ID)
	fmt.Printf("Image %d is being transferred to region '%s'.\n", imageID, region)
	return nil
}
