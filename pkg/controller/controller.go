package controller

import (
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/sw33tLie/fleex/pkg/box"
	"github.com/sw33tLie/fleex/pkg/digitalocean"
	"github.com/sw33tLie/fleex/pkg/linode"
	"github.com/sw33tLie/fleex/pkg/utils"
)

type Provider int

const (
	PROVIDER_LINODE       = 1
	PROVIDER_DIGITALOCEAN = 2
)

const (
	INVALID_PROVIDER = "Invalid Provider!"
)

var log = logrus.New()

func GetProvider(name string) Provider {
	name = strings.ToLower(name)

	switch name {
	case "linode":
		return PROVIDER_LINODE
	case "digitalocean":
		return PROVIDER_DIGITALOCEAN
	}

	return -1
}

// ListBoxes prints all active boxes of a provider
func ListBoxes(token string, provider Provider) {
	switch provider {
	case PROVIDER_LINODE:
		linode.ListBoxes(token)
	case PROVIDER_DIGITALOCEAN:
		digitalocean.ListBoxes(token)
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
	}
}

// DeleteFleet deletes a whole fleet or a single box
func DeleteFleet(name string, token string, provider Provider) {
	switch provider {
	case PROVIDER_LINODE:
		linode.DeleteFleet(name, token)
	case PROVIDER_DIGITALOCEAN:
		digitalocean.DeleteFleet(name, token)
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
	}
}

// ListImages prints a list of available private images of a provider
func ListImages(token string, provider Provider) {
	switch provider {
	case PROVIDER_LINODE:
		linode.ListImages(token)
	case PROVIDER_DIGITALOCEAN:
		digitalocean.ListImages(token)
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
	}
}

func GetFleet(fleetName string, token string, provider Provider) []box.Box {
	switch provider {
	case PROVIDER_LINODE:
		return linode.GetFleet(fleetName, token)
	case PROVIDER_DIGITALOCEAN:
		// return digitalocean.GetFleet(fleetName, token)
		return nil
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
		return nil
	}
}

func RunCommand(name string, command string, token string, provider Provider) {
	switch provider {
	case PROVIDER_LINODE:
		linode.RunCommand(name, command, token)
	case PROVIDER_DIGITALOCEAN:
		// TODO
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
	}
}

func DeleteBoxByID(id int, token string, provider Provider) {
	switch provider {
	case PROVIDER_LINODE:
		linode.DeleteBoxByID(id, token)
	case PROVIDER_DIGITALOCEAN:
		// TODO
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
	}
}

func SpawnFleet(fleetName string, fleetCount int, image string, region string, token string, wait bool, provider Provider) {
	switch provider {
	case PROVIDER_LINODE:
		linode.SpawnFleet(fleetName, fleetCount, image, region, token, wait)
	case PROVIDER_DIGITALOCEAN:
		digitalocean.SpawnFleet(fleetName, fleetCount, region, token)
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
	}
}

func SSH(boxName string, token string, provider Provider) {
	switch provider {
	case PROVIDER_LINODE:
		linode.SSH(boxName, token)
	case PROVIDER_DIGITALOCEAN:
		// TODO
	default:
		utils.Log.Fatal(INVALID_PROVIDER)
	}
}
