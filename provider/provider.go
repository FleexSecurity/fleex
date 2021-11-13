package provider

import (
	"errors"

	"github.com/FleexSecurity/fleex/pkg/box"
	"github.com/linode/linodego"
)

var (
	ErrQuery    = errors.New("error on query")
	ErrNotFound = errors.New("not found")
)

type Service interface {
	SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, token string)
	GetClient(token string) linodego.Client
	GetBoxes(token string) (boxes []box.Box)
	GetFleet(fleetName, token string) (fleet []box.Box)
	GetBox(boxName, token string) box.Box
	GetImages(token string) (images []box.Image)
	ListBoxes(token string)
	ListImages(token string)
	RunCommand(name, command string, port int, username, password, token string)
	CountFleet(fleetName string, boxes []box.Box) (count int)
	DeleteFleet(name string, token string)
	DeleteBoxByID(id string, token string)
	DeleteBoxByLabel(label string, token string)
	SpawnBox(name string, image string, region string, size string, token string)
	CreateImage(token string, linodeID int, label string)
	GetDiskID(token string, linodeID int) int
}
