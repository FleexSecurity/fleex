package digitalocean

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/FleexSecurity/fleex/pkg/box"
	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/digitalocean/godo"
	"github.com/spf13/viper"
)

// SpawnFleet spawns a DigitalOcean fleet
func SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, sshFingerprint string, tags []string, token string) {
	existingFleet := GetFleet(fleetName, token)

	client := godo.NewFromToken(token)
	ctx := context.TODO()
	digitaloceanPasswd := viper.GetString("digitalocean.password")
	if digitaloceanPasswd == "" {
		digitaloceanPasswd = "1337rootPass"
	}

	droplets := []string{}

	for i := 0; i < fleetCount; i++ {
		droplets = append(droplets, fleetName+"-"+strconv.Itoa(i+1+len(existingFleet)))
	}

	user_data := `#!/bin/bash
sudo sed -i "/^[^#]*PasswordAuthentication[[:space:]]no/c\PasswordAuthentication yes" /etc/ssh/sshd_config
sudo service sshd restart
echo 'op:` + digitaloceanPasswd + `' | sudo chpasswd`

	var createRequest *godo.DropletMultiCreateRequest
	imageIntID, err := strconv.Atoi(image)
	if err != nil {
		createRequest = &godo.DropletMultiCreateRequest{
			Names:  droplets,
			Region: region,
			Size:   size,
			// UserData: "echo 'root:" + digitaloceanPasswd + "' | chpasswd",
			UserData: user_data,
			Image: godo.DropletCreateImage{
				Slug: image,
			},
			SSHKeys: []godo.DropletCreateSSHKey{
				{Fingerprint: sshFingerprint},
			},
			Tags: tags,
		}
	} else {
		createRequest = &godo.DropletMultiCreateRequest{
			Names:    droplets,
			Region:   region,
			Size:     size,
			UserData: user_data,
			Image: godo.DropletCreateImage{
				ID: imageIntID,
			},
			SSHKeys: []godo.DropletCreateSSHKey{
				{Fingerprint: sshFingerprint},
			},
			Tags: tags,
		}
	}

	_, _, err = client.Droplets.CreateMultiple(ctx, createRequest)

	if err != nil {
		utils.Log.Fatal(err)
	}

}

func GetFleet(fleetName, token string) (fleet []box.Box) {
	boxes := GetBoxes(token)

	for _, box := range boxes {
		if strings.HasPrefix(box.Label, fleetName) {
			fleet = append(fleet, box)
		}
	}
	return fleet
}

// GetBox returns a single box by its label
func GetBox(boxName, token string) box.Box {
	boxes := GetBoxes(token)

	for _, box := range boxes {
		if box.Label == boxName {
			return box
		}
	}
	utils.Log.Fatal("Box not found!")
	return box.Box{}
}

func GetBoxes(token string) (boxes []box.Box) {
	client := godo.NewFromToken(token)
	ctx := context.TODO()
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 9999,
	}

	droplets, _, err := client.Droplets.List(ctx, opt)
	if err != nil {
		utils.Log.Fatal(err)
	}

	for _, d := range droplets {
		ip, _ := d.PublicIPv4()
		dID := strconv.Itoa(d.ID)
		boxes = append(boxes, box.Box{ID: dID, Label: d.Name, Group: "", Status: d.Status, IP: ip})
	}
	return boxes
}

func ListBoxes(token string) {
	boxes := GetBoxes(token)
	for _, box := range boxes {
		fmt.Println(box.ID, box.Label, box.Group, box.Status, box.IP)
	}
}

func DeleteFleet(name string, token string) {
	droplets := GetBoxes(token)
	for _, droplet := range droplets {
		if droplet.Label == name {
			// It's a single box
			DeleteBoxByID(droplet.ID, token)
			return
		}
	}

	// Otherwise, we got a fleet to delete
	for _, droplet := range droplets {
		if strings.HasPrefix(droplet.Label, name) {
			DeleteBoxByID(droplet.ID, token)
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
		utils.Log.Fatal(err)
	}
	for _, image := range images {
		fmt.Println(image.ID, image.Name, image.Status, image.SizeGigaBytes)
	}
}

func DeleteBoxByID(ID string, token string) {
	client := godo.NewFromToken(token)
	ctx := context.TODO()

	ID1, _ := strconv.Atoi(ID)
	_, err := client.Droplets.Delete(ctx, ID1)
	if err != nil {
		utils.Log.Fatal(err)
	}
}

func deleteBoxByTag(tag string, token string) {
	client := godo.NewFromToken(token)
	ctx := context.TODO()

	_, err := client.Droplets.DeleteByTag(ctx, tag)
	if err != nil {
		utils.Log.Fatal(err)
	}
}

func CountFleet(fleetName string, boxes []box.Box) (count int) {
	for _, box := range boxes {
		if strings.HasPrefix(box.Label, fleetName) {
			count++
		}
	}
	return count
}

func RunCommand(name, command string, port int, username, password, token string) {
	//doSshUser := viper.GetString("digitalocean.username")
	//doSshPort := viper.GetInt("digitalocean.port")
	// doSshPassword := viper.GetString("digitalocean.password")
	boxes := GetBoxes(token)

	// fmt.Println(port, username, password)

	for _, box := range boxes {
		if box.Label == name {
			// It's a single box
			boxIP := box.IP
			sshutils.RunCommand(command, boxIP, port, username, password)
			return
		}
	}

	// Otherwise, send command to a fleet
	fleetSize := CountFleet(name, boxes)

	fleet := make(chan *box.Box, fleetSize)
	processGroup := new(sync.WaitGroup)
	processGroup.Add(fleetSize)

	for i := 0; i < fleetSize; i++ {
		go func() {
			for {
				box := <-fleet

				if box == nil {
					break
				}
				boxIP := box.IP
				sshutils.RunCommand(command, boxIP, port, username, password)
			}
			processGroup.Done()
		}()
	}

	for i := range boxes {
		if strings.HasPrefix(boxes[i].Label, name) {
			fleet <- &boxes[i]
		}
	}

	close(fleet)
	processGroup.Wait()
}

func RunCommandByIP(ip, command string, port int, username, password, token string) {
	sshutils.RunCommand(command, ip, port, username, password)
}

func CreateImage(token string, diskID int, label string) {
	client := godo.NewFromToken(token)
	ctx := context.TODO()

	_, _, err := client.DropletActions.Snapshot(ctx, diskID, label)
	if err != nil {
		utils.Log.Fatal(err)
	}
}
