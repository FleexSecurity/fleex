package controller

import (
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/sw33tLie/fleex/pkg/digitalocean"
	"github.com/sw33tLie/fleex/pkg/linode"
)

type Provider int

const (
	PROVIDER_LINODE       = 1
	PROVIDER_DIGITALOCEAN = 2
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
		log.Fatal("Invalid Provider")
	}
}
