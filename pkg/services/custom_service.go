package services

import (
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/provider"
)

type CustomService struct {
	Configs *models.Config
}

// SpawnFleet spawns a Vultr fleet
func (c CustomService) SpawnFleet(fleetName string, fleetCount int) error {
	return nil
}

// GetBoxes returns a slice containg all active boxes of a Linode account
func (c CustomService) GetBoxes() (boxes []provider.Box, err error) {
	return boxes, nil
}

// GetBoxes returns a slice containg all boxes of a given fleet
func (c CustomService) GetFleet(fleetName string) (fleet []provider.Box, err error) {
	return fleet, nil
}

// GetBox returns a single box by its label
func (c CustomService) GetBox(boxName string) (provider.Box, error) {
	return provider.Box{}, provider.ErrBoxNotFound
}

// GetImages returns a slice containing all snapshots of vultr account
func (c CustomService) GetImages() (images []provider.Image) {
	return []provider.Image{}
}

// ListBoxes prints all active boxes of a vultr account
func (c CustomService) ListBoxes() {
}

// ListImages prints snapshots of vultr account
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
