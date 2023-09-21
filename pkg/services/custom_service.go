package services

import (
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/provider"
)

type CustomService struct {
	Configs *models.Config
}

func (c CustomService) SpawnFleet(fleetName string, fleetCount int) error {
	return nil
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
	return fleet, nil
}

func (c CustomService) GetBox(boxName string) (provider.Box, error) {
	return provider.Box{}, models.ErrBoxNotFound
}

func (c CustomService) GetImages() (images []provider.Image) {
	return []provider.Image{}
}

func (c CustomService) ListImages() error {
	return nil
}

func (c CustomService) RemoveImages(name string) error {
	return nil
}

func (c CustomService) DeleteFleet(name string) error {
	return nil
}

func (c CustomService) DeleteBoxByID(id string) error {
	return nil
}

func (c CustomService) DeleteBoxByLabel(label string) error {
	return nil
}

func (c CustomService) RunCommand(name, command string, port int, username, password string) error {
	return nil
}

func (c CustomService) CountFleet(fleetName string, boxes []provider.Box) (count int) {
	return count
}

func (c CustomService) spawnBox(name string) error {
	return nil
}

func (c CustomService) CreateImage(diskID int, label string) error {
	return nil
}

func (c CustomService) getSSHKey() string {
	return ""
}

func (c CustomService) KeyCheck(fleex_key string) string {
	return ""
}
