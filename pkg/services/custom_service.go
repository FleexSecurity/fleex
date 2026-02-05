package services

import (
	"fmt"
	"sync"

	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/provider"
	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/utils"
)

type CustomService struct {
	Configs *models.Config
}

func (c CustomService) SpawnFleet(fleetName string, fleetCount int) error {
	return models.ErrNotAvailableCustomVps
}

func (c CustomService) GetBoxes() (boxes []provider.Box, err error) {
	customVps := c.Configs.CustomVMs

	for _, vps := range customVps {
		boxes = append(boxes, provider.Box{
			ID:     vps.InstanceID,
			Label:  vps.InstanceID,
			Group:  "custom",
			Status: "unknown",
			IP:     vps.PublicIP,
		})
	}
	return boxes, nil
}

func (c CustomService) GetFleet(fleetName string) (fleet []provider.Box, err error) {
	boxes, err := c.GetBoxes()
	if err != nil {
		return []provider.Box{}, err
	}

	for _, box := range boxes {
		if utils.MatchesFleetName(box.ID, fleetName) {
			fleet = append(fleet, box)
		}
	}
	return fleet, nil
}

func (c CustomService) GetBox(boxName string) (provider.Box, error) {
	boxes, err := c.GetBoxes()
	if err != nil {
		return provider.Box{}, err
	}

	for _, box := range boxes {
		if utils.MatchesFleetName(box.ID, boxName) {
			return box, nil
		}
	}
	return provider.Box{}, models.ErrBoxNotFound
}

func (c CustomService) GetImages() (images []provider.Image, err error) {
	return []provider.Image{}, nil
}

func (c CustomService) ListImages() error {
	return models.ErrNotAvailableCustomVps
}

func (c CustomService) RemoveImages(name string) error {
	return models.ErrNotAvailableCustomVps
}

func (c CustomService) DeleteFleet(name string) error {
	return models.ErrNotAvailableCustomVps
}

func (c CustomService) DeleteBoxByID(id string) error {
	return models.ErrNotAvailableCustomVps
}

func (c CustomService) DeleteBoxByLabel(label string) error {
	return models.ErrNotAvailableCustomVps
}

func (c CustomService) RunCommand(name, command string, port int, username, password string) error {
	for _, box := range c.Configs.CustomVMs {
		if utils.MatchesFleetName(box.InstanceID, name) {
			sshutils.RunCommand(command, box.PublicIP, box.SSHPort, box.Username, c.Configs.SSHKeys.PrivateFile)
			return nil
		}
	}

	fleetSize := len(c.Configs.CustomVMs)

	if fleetSize == 0 {
		return fmt.Errorf("No boxes with name %s", name)
	}

	var wg sync.WaitGroup
	wg.Add(fleetSize)

	for _, box := range c.Configs.CustomVMs {
		go func(b models.CustomVM) {
			defer wg.Done()
			sshutils.RunCommand(command, b.PublicIP, b.SSHPort, b.Username, c.Configs.SSHKeys.PrivateFile)
		}(box)
	}

	wg.Wait()
	return nil
}

func (c CustomService) CountFleet(fleetName string, boxes []provider.Box) (count int) {
	return len(c.Configs.CustomVMs)
}

func (c CustomService) CreateImage(diskID int, label string) error {
	return models.ErrNotAvailableCustomVps
}

func (c CustomService) TransferImage(imageID int, region string) error {
	return models.ErrNotAvailableCustomVps
}

func (c CustomService) GetImageRegions(imageID int) ([]string, error) {
	return nil, models.ErrNotAvailableCustomVps
}
