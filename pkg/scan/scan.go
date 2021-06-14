package scan

import (
	"bufio"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hnakamur/go-scp"

	"github.com/sw33tLie/fleex/pkg/box"
	"github.com/sw33tLie/fleex/pkg/controller"
	"github.com/sw33tLie/fleex/pkg/sshutils"
	"github.com/sw33tLie/fleex/pkg/utils"
)

// Start runs a scan
func Start(fleetName, command string, delete bool, input, output, token string, port int, username, password string, provider controller.Provider) {
	start := time.Now()
	// Make local temp folder
	tempFolder := path.Join("/tmp", strconv.FormatInt(time.Now().UnixNano(), 10))
	tempFolderInput := path.Join(tempFolder, "input")
	tempFolderOutput := path.Join(tempFolder, "output")
	// Create temp folder
	utils.MakeFolder(tempFolder)
	utils.MakeFolder(tempFolderInput)
	utils.MakeFolder(tempFolderOutput)
	utils.Log.Info("Scan started. Output folder: ", tempFolderOutput)

	// Input file to string
	inputString := utils.FileToString(input)

	fleet := controller.GetFleet(fleetName, token, provider)
	if len(fleet) < 1 {
		utils.Log.Fatal("No fleet found")
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
			chunkPath := path.Join(tempFolderInput, "chunk-"+fleetName+"-"+strconv.Itoa(counter/linesPerChunk))
			utils.StringToFile(chunkPath, chunkContent)
			inputFiles = append(inputFiles, chunkPath)
			chunkContent = ""
		}
		counter++
	}

	if scanner.Err() != nil {
		utils.Log.Fatal(scanner.Err())
	}

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
				err := scp.NewSCP(sshutils.GetConnection(l.IP, port, username, password).Client).SendFile(path.Join(tempFolderInput, "chunk-"+boxName), "/home/op")
				if err != nil {
					utils.Log.Fatal("Failed to send file: ", err)
				}

				// Replace labels and craft final command
				finalCommand := command
				finalCommand = strings.ReplaceAll(finalCommand, "{{INPUT}}", path.Join("/home/op", "chunk-"+boxName))
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
