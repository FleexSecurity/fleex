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

type Service interface {
	SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, sshFingerprint string, tags []string, token string) error
	GetBoxes(token string) (boxes []Box, err error)
	GetFleet(fleetName, token string) (fleet []Box, err error)
	GetBox(boxName, token string) (Box, error)
	ListBoxes(token string)
	ListImages(token string) error
	RunCommand(name, command string, port int, username, password, token string) error
	CountFleet(fleetName string, boxes []Box) (count int)
	DeleteFleet(name string, token string) error
	DeleteBoxByID(id string, token string) error
	DeleteBoxByLabel(label string, token string) error
	CreateImage(token string, diskID int, label string) error
}
