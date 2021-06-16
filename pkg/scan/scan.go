package scan

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hnakamur/go-scp"
	"github.com/mitchellh/go-homedir"

	"github.com/sw33tLie/fleex/pkg/box"
	"github.com/sw33tLie/fleex/pkg/controller"
	"github.com/sw33tLie/fleex/pkg/sshutils"
	"github.com/sw33tLie/fleex/pkg/utils"
)

func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

func GetLine(filename string, names chan string, readerr chan error) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		names <- scanner.Text()
	}
	readerr <- scanner.Err()
}

// Start runs a scan
func Start(fleetName, command string, delete bool, input, output, token string, port int, username, password string, provider controller.Provider) {
	start := time.Now()

	// Get home dir
	homeDir, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	outputFolder := path.Join(homeDir, "fleex")

	if output != "" {
		outputFolder = output
	}

	// Make local temp folder
	tempFolder := path.Join(outputFolder, strconv.FormatInt(time.Now().UnixNano(), 10))
	tempFolderInput := path.Join(tempFolder, "input")
	tempFolderOutput := path.Join(tempFolder, "output")
	// Create temp folder
	utils.MakeFolder(tempFolder)
	utils.MakeFolder(tempFolderInput)
	utils.MakeFolder(tempFolderOutput)
	utils.Log.Info("Scan started. Output folder: ", tempFolderOutput)

	// Input file to string

	fleet := controller.GetFleet(fleetName, token, provider)
	if len(fleet) < 1 {
		utils.Log.Fatal("No fleet found")
	}

	/////

	// First get lines count
	file, err := os.Open(input)

	if err != nil {
		utils.Log.Fatal(err)
	}

	linesCount, err := lineCounter(file)

	if err != nil {
		utils.Log.Fatal(err)
	}

	fmt.Println("Fleet count: ", len(fleet))
	linesPerChunk := linesCount / len(fleet)
	linesPerChunkRest := linesCount % len(fleet)

	names := make(chan string)
	readerr := make(chan error)

	go GetLine(input, names, readerr)
	counter := 1
	asd := []string{}

	x := 1
loop:
	for {
		select {
		case name := <-names:
			// Process each line
			asd = append(asd, name)

			re := 0

			if linesPerChunkRest > 0 {
				re = 1
			}
			if counter%(linesPerChunk+re) == 0 {
				utils.StringToFile(path.Join(tempFolderInput, "chunk-"+fleetName+"-"+strconv.Itoa(x)), strings.Join(asd[0:counter], "\n")+"\n")
				asd = nil
				x++
				counter = 0
				linesPerChunkRest--

			}
			counter++

		case err := <-readerr:
			if err != nil {
				utils.Log.Fatal(err)
			}
			break loop
		}
	}

	utils.Log.Debug("Generated file chunks")

	/////

	fleetNames := make(chan *box.Box, len(fleet))
	processGroup := new(sync.WaitGroup)
	processGroup.Add(len(fleet))

	for i := 0; i < len(fleet); i++ {
		go func() {
			for {
				l := <-fleetNames

				if l == nil {
					break
				}

				boxName := l.Label

				// Send input file via SCP
				err := scp.NewSCP(sshutils.GetConnection(l.IP, port, username, password).Client).SendFile(path.Join(tempFolderInput, "chunk-"+boxName), "/home/"+username)
				if err != nil {
					utils.Log.Fatal("Failed to send file: ", err)
				}

				// Replace labels and craft final command
				finalCommand := command
				finalCommand = strings.ReplaceAll(finalCommand, "{{INPUT}}", path.Join("/home/"+username, "chunk-"+boxName))
				finalCommand = strings.ReplaceAll(finalCommand, "{{OUTPUT}}", "chunk-res-"+boxName)

				sshutils.RunCommand(finalCommand, l.IP, port, username, password)

				// Now download the output file
				err = scp.NewSCP(sshutils.GetConnection(l.IP, port, username, password).Client).ReceiveFile("chunk-res-"+boxName, path.Join(tempFolderOutput, "chunk-res-"+boxName))
				if err != nil {
					utils.Log.Fatal("Failed to get file: ", err)
				}

				if delete {
					// TODO: Not the best way to delete a box, if this program crashes/is stopped
					// before reaching this line the box won't be deleted. It's better to setup
					// a cron/command on the box directly.
					controller.DeleteBoxByID(l.ID, token, provider)
					utils.Log.Debug("Killed box ", l.Label)
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
	duration := time.Since(start)
	utils.Log.Info("Scan done! Took ", duration, " seconds")
}
