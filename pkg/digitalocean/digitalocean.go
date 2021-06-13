package digitalocean

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/box"
)

// SpawnFleet spawns a DigitalOcean fleet
func SpawnFleet(fleetName string, fleetCount int, region string, size string, slug string, token string) {
	fmt.Println("Digitalocean Spawn", token)
	digSsh := viper.GetString("digitalocean.ssh-fingerprint")
	digTags := viper.GetStringSlice("digitalocean.tags")

	client := godo.NewFromToken(token)
	ctx := context.TODO()

	droplets := []string{}

	for i := 0; i < fleetCount; i++ {
		droplets = append(droplets, fleetName+strconv.Itoa(i+1))
	}

	createRequest := &godo.DropletMultiCreateRequest{
		Names:  droplets,
		Region: region,
		Size:   size,
		/*Image: godo.DropletCreateImage{
			ID: IDIMAGE,
		},*/
		Image: godo.DropletCreateImage{
			Slug: slug,
		},
		SSHKeys: []godo.DropletCreateSSHKey{
			godo.DropletCreateSSHKey{Fingerprint: digSsh},
		},
		Tags: digTags,
	}

	_, _, err := client.Droplets.CreateMultiple(ctx, createRequest)

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

}

func GetFleet(fleetName, token string) []box.Box {
	// TODO
	return nil
}

func GetBoxes(token string) []godo.Droplet {
	client := godo.NewFromToken(token)
	ctx := context.TODO()
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 9999,
	}

	droplets, _, err := client.Droplets.List(ctx, opt)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	return droplets
}

func ListBoxes(token string) {
	droplets := GetBoxes(token)
	for _, drop := range droplets {
		fmt.Println(drop.ID, "-", drop.Name, "-", drop.Size.Disk, "gb - ", drop.Status, "-", drop.Created)
	}
}

func DeleteFleet(name string, token string) {
	droplets := GetBoxes(token)
	for _, droplet := range droplets {
		if droplet.Name == name {
			// It's a single box
			deleteBoxByID(droplet.ID, token)
			return
		}
	}

	// Otherwise, we got a fleet to delete
	for _, droplet := range droplets {
		fmt.Println(droplet.Name, name)
		if strings.HasPrefix(droplet.Name, name) {
			deleteBoxByID(droplet.ID, token)
		}
	}
}

func ListImages(token string) {
	// TODO
	client := godo.NewFromToken(token)
	ctx := context.TODO()
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 9999,
	}

	images, _, err := client.Images.ListUser(ctx, opt)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	for _, image := range images {
		fmt.Println(image.ID, image.Name, image.Status, image.SizeGigaBytes)
	}
}

func deleteBoxByID(ID int, token string) {
	client := godo.NewFromToken(token)
	ctx := context.TODO()

	_, err := client.Droplets.Delete(ctx, ID)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return
	}
}

func deleteBoxByTag(tag string, token string) {
	client := godo.NewFromToken(token)
	ctx := context.TODO()

	_, err := client.Droplets.DeleteByTag(ctx, tag)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return
	}
}
