package services

import (
	"fmt"
	"strings"
	"sync"

	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/provider"
	"github.com/FleexSecurity/fleex/pkg/sshutils"
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
			Label:  vps.Provider,
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
		if strings.HasPrefix(box.ID, fleetName) {
			fleet = append(fleet, box)
		}
	}
	return fleet, nil
}

func (c CustomService) GetBox(boxName string) (provider.Box, error) {
	return provider.Box{}, models.ErrBoxNotFound
}

func (c CustomService) GetImages() (images []provider.Image) {
	return []provider.Image{}
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
		if strings.HasPrefix(box.InstanceID, name) {
			sshutils.RunCommandWithPassword(command, box.PublicIP, box.SSHPort, box.Username, box.Password)
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
			sshutils.RunCommandWithPassword(command, b.PublicIP, b.SSHPort, b.Username, b.Password)
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
