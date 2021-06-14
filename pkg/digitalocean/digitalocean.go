package digitalocean

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/digitalocean/godo"
	"github.com/hnakamur/go-scp"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/sshutils"
	"github.com/sw33tLie/fleex/pkg/utils"
)

// SpawnFleet spawns a DigitalOcean fleet
func SpawnFleet(fleetName string, fleetCount int, region string, token string) {
	// fmt.Println("Digitalocean Spawn", token)
	digSsh := viper.GetString("digitalocean.ssh-fingerprint")
	digTags := viper.GetStringSlice("digitalocean.tags")
	digSize := viper.GetString("digitalocean.size")
	digImageID := viper.GetInt("digitalocean.image-id")
	fmt.Println(digImageID)

	client := godo.NewFromToken(token)
	ctx := context.TODO()

	droplets := []string{}

	for i := 0; i < fleetCount; i++ {
		droplets = append(droplets, fleetName+strconv.Itoa(i+1))
	}

	createRequest := &godo.DropletMultiCreateRequest{
		Names:  droplets,
		Region: region,
		Size:   digSize,
		Image: godo.DropletCreateImage{
			ID: digImageID,
		},
		/*Image: godo.DropletCreateImage{
			Slug: slug,
		},*/
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

func GetFleet(fleetName, token string) (fleet []godo.Droplet) {
	boxes := GetBoxes(token)

	for _, box := range boxes {
		if strings.HasPrefix(box.Name, fleetName) {
			fleet = append(fleet, box)
		}
	}
	return fleet
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

func CountFleet(fleetName string, boxes []godo.Droplet) (count int) {
	for _, box := range boxes {
		if strings.HasPrefix(box.Name, fleetName) {
			count++
		}
	}
	return count
}

func RunCommand(name string, command string, token string, port int, username string, password string) {
	doSshUser := viper.GetString("digitalocean.username")
	doSshPort := viper.GetInt("digitalocean.port")
	doSshPassword := viper.GetString("digitalocean.password")
	boxes := GetBoxes(token)
	// fmt.Println(boxes)
	for _, box := range boxes {
		if box.Name == name {
			// It's a single box
			boxIP := box.Networks.V4[1].IPAddress
			//fmt.Println(boxIP)
			sshutils.RunCommand(command, boxIP, port, username, password)
			return
		}
	}

	// Otherwise, send command to a fleet
	fleetSize := CountFleet(name, boxes)

	fleet := make(chan *godo.Droplet, fleetSize)
	processGroup := new(sync.WaitGroup)
	processGroup.Add(fleetSize)

	for i := 0; i < fleetSize; i++ {
		go func() {
			for {
				box := <-fleet

				if box == nil {
					break
				}
				boxIP := box.Networks.V4[1].IPAddress
				sshutils.RunCommand(command, boxIP, doSshPort, doSshUser, doSshPassword)
			}
			processGroup.Done()
		}()
	}

	for i := range boxes {
		if strings.HasPrefix(boxes[i].Name, name) {
			fleet <- &boxes[i]
		}
	}

	close(fleet)
	processGroup.Wait()
}

func Scan(fleetName string, command string, delete bool, input string, output string, token string) {
	doSshUser := viper.GetString("digitalocean.username")
	doSshPort := viper.GetInt("digitalocean.port")
	doSshPassword := viper.GetString("digitalocean.password")

	// Make local temp folder
	tempFolder := path.Join("/tmp", strconv.FormatInt(time.Now().UnixNano(), 10))

	// Create temp folder
	err := os.Mkdir(tempFolder, 0755)
	if err != nil {
		log.Fatal(err)
	}

	// Input file to string
	inputString := utils.FileToString(input)

	fleet := GetFleet(fleetName, token)

	if len(fleet) < 1 {
		log.Fatal("No fleet found")
	}

	linesCount := utils.LinesCount(inputString)
	linesPerChunk := linesCount / len(fleet)

	// Iterate over multiline input string
	scanner := bufio.NewScanner(strings.NewReader(inputString))

	counter := 1
	chunkContent := ""
	var inputFiles []string
	for scanner.Scan() {
		line := scanner.Text()
		chunkContent += line + "\n"
		if counter%linesPerChunk == 0 {
			// Remove bottom empty line
			chunkContent = strings.TrimSuffix(chunkContent, "\n")
			// Save chunk
			chunkPath := path.Join(tempFolder, "chunk-"+fleetName+strconv.Itoa(counter/linesPerChunk))
			utils.StringToFile(chunkPath, chunkContent)
			inputFiles = append(inputFiles, chunkPath)

			fmt.Println("LL"+chunkPath, inputFiles)

			chunkContent = ""
		}
		counter++
	}

	if scanner.Err() != nil {
		log.Println(scanner.Err())
	}

	// Send SSH commands to all boxes

	fleetNames := make(chan *godo.Droplet, len(fleet))
	processGroup := new(sync.WaitGroup)
	processGroup.Add(len(fleet))

	for i := 0; i < len(fleet); i++ {
		go func() {
			for {
				do := <-fleetNames

				if do == nil {
					break
				}

				doName := do.Name

				// Send input file via SCP
				err := scp.NewSCP(sshutils.GetConnection(do.Networks.V4[1].IPAddress, doSshPort, doSshUser, doSshPassword).Client).SendFile(path.Join(tempFolder, "chunk-"+doName), "/home/op")
				if err != nil {
					log.Fatalf("Failed to send file: %s", err)
				}

				// Replace labels and craft final command
				finalCommand := command
				finalCommand = strings.ReplaceAll(finalCommand, "{{INPUT}}", path.Join("/home/op", "chunk-"+doName))
				finalCommand = strings.ReplaceAll(finalCommand, "{{OUTPUT}}", "chunk-res-"+doName)

				fmt.Println("SCANNING WITH ", path.Join(tempFolder, "chunk-"+doName), " ")
				// TODO: Not optimal, it runs GetBoxes() every time which is dumb, should use a function that does the same but by id
				fmt.Println(finalCommand)
				RunCommand(doName, finalCommand, token, doSshPort, doSshUser, doSshPassword)

				// Now download the output file
				err = scp.NewSCP(sshutils.GetConnection(do.Networks.V4[1].IPAddress, doSshPort, doSshUser, doSshPassword).Client).ReceiveFile("chunk-res-"+doName, path.Join(tempFolder, "chunk-res-"+doName))
				if err != nil {
					log.Fatalf("Failed to get file: %s", err)
				}

				if delete {
					// TODO: Not the best way to delete a box, if this program crashes/is stopped
					// before reaching this line the box won't be deleted. It's better to setup
					// a cron/command on the box directly.
					deleteBoxByID(do.ID, token)
				}

			}
			processGroup.Done()
		}()
	}

	for i := range fleet {
		fleetNames <- &fleet[i]
	}

	close(fleetNames)
	processGroup.Wait()

	// Scan done, process results

	fmt.Println("SCAN DONE")
}
