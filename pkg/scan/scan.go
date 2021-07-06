package scan

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
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
func Start(fleetName, command string, delete bool, input, outputPath, chunksFolder string, token string, port int, username, password string, provider controller.Provider) {
	start := time.Now()

	timeStamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	// TODO: use a proper temp folder function so that it can run on windows too
	tempFolder := filepath.Join("/tmp", timeStamp)

	if chunksFolder != "" {
		tempFolder = chunksFolder
	}

	// Make local temp folder
	tempFolderInput := filepath.Join(tempFolder, "input")
	// Create temp folder
	utils.MakeFolder(tempFolder)
	utils.MakeFolder(tempFolderInput)
	utils.Log.Info("Scan started!")

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

	utils.Log.Debug("Fleet count: ", len(fleet))
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
				utils.StringToFile(filepath.Join(tempFolderInput, "chunk-"+fleetName+"-"+strconv.Itoa(x)), strings.Join(asd[0:counter], "\n")+"\n")
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
				err := scp.NewSCP(sshutils.GetConnection(l.IP, port, username, password).Client).SendFile(filepath.Join(tempFolderInput, "chunk-"+boxName), "/tmp/fleex-"+timeStamp+"-chunk-"+boxName)
				if err != nil {
					utils.Log.Fatal("Failed to send file: ", err)
				}

				chunkInputFile := "/tmp/fleex-" + timeStamp + "-chunk-" + boxName
				chunkOutputFile := "/tmp/fleex-" + timeStamp + "-chunk-out-" + boxName

				// Replace labels and craft final command
				finalCommand := command
				finalCommand = strings.ReplaceAll(finalCommand, "{{INPUT}}", chunkInputFile)
				finalCommand = strings.ReplaceAll(finalCommand, "{{OUTPUT}}", chunkOutputFile)

				sshutils.RunCommand(finalCommand, l.IP, port, username, password)

				// Now download the output file
				err = scp.NewSCP(sshutils.GetConnection(l.IP, port, username, password).Client).ReceiveFile(chunkOutputFile, filepath.Join(tempFolder, "chunk-out-"+boxName))
				if err != nil {
					utils.Log.Fatal("Failed to get file: ", err)
				}

				// Remove input chunk file from remote box to save space
				sshutils.RunCommand("sudo rm -rf "+chunkInputFile+" "+chunkOutputFile, l.IP, port, username, password)

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
	utils.Log.Info("Scan done! Took ", duration, ". Output file: ", outputPath)

	// Remove input folder when the scan is done
	os.RemoveAll(tempFolderInput)

	// TODO: Get rid of bash and do this using Go

	utils.RunCommand("cat " + filepath.Join(tempFolder, "*") + " > " + outputPath)

	if chunksFolder == "" {
		//utils.RunCommand("rm -rf " + filepath.Join(tempFolder, "chunk-out-*"))
		os.RemoveAll(tempFolder)
	}

}
