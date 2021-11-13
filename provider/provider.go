package provider

import (
	"errors"
)

var (
	ErrQuery    = errors.New("error on query")
	ErrNotFound = errors.New("not found")
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
	SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, sshFingerprint string, tags []string, token string)
	GetBoxes(token string) (boxes []Box)
	GetFleet(fleetName, token string) (fleet []Box)
	GetBox(boxName, token string) Box
	ListBoxes(token string)
	ListImages(token string)
	RunCommand(name, command string, port int, username, password, token string)
	CountFleet(fleetName string, boxes []Box) (count int)
	DeleteFleet(name string, token string)
	DeleteBoxByID(id string, token string)
	DeleteBoxByLabel(label string, token string)
	CreateImage(token string, diskID int, label string)
}
