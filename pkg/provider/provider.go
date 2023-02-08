package provider

import (
	"errors"
)

var (
	ErrGeneric         = errors.New("something went wrong, check that the data in the config.yaml is correct")
	ErrInvalidProvider = errors.New("invalid provider")
	ErrInvalidToken    = errors.New("invalid token")
	ErrInvalidImage    = errors.New("invalid image")
	ErrInvalidRegion   = errors.New("invalid region")
	ErrInvalidSize     = errors.New("invalid size")
	ErrInvalidPort     = errors.New("invalid port")
	ErrInvalidSshFile  = errors.New("invalid SSH file")
	ErrBoxNotFound     = errors.New("box not found")
)

type Box struct {
	ID     string
	Label  string
	Group  string
	Status string
	IP     string
}

type Image struct {
	ID      string
	Label   string
	Created string
	Size    int
	Vendor  string
}

type Provider interface {
	SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, sshFingerprint string, tags []string) error
	GetBoxes() (boxes []Box, err error)
	GetFleet(fleetName string) (fleet []Box, err error)
	GetBox(boxName string) (Box, error)
	ListBoxes()
	ListImages() error
	RemoveImages(name string) error
	RunCommand(name, command string, port int, username, password string) error
	CountFleet(fleetName string, boxes []Box) (count int)
	DeleteFleet(name string) error
	DeleteBoxByID(id string) error
	DeleteBoxByLabel(label string) error
	CreateImage(diskID int, label string) error
}
