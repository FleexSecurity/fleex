package controller

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hnakamur/go-scp"

	"github.com/FleexSecurity/fleex/pkg/models"
	p "github.com/FleexSecurity/fleex/pkg/provider"
	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/utils"
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

func ReplaceCommandVars(command string, vars map[string]string) (string, error) {
	if _, ok := vars["INPUT"]; !ok {
		return "", fmt.Errorf("missing 'INPUT' variable")
	}
	if _, ok := vars["OUTPUT"]; !ok {
		return "", fmt.Errorf("missing 'OUTPUT' variable")
	}

	for key, value := range vars {
		placeholder := fmt.Sprintf("{vars.%s}", key)
		command = strings.ReplaceAll(command, placeholder, value)
	}
	return command, nil
}

func ReplaceVerticalCommandVars(command string, vars map[string]string) (string, error) {
	if _, ok := vars["OUTPUT"]; !ok {
		return "", fmt.Errorf("missing 'OUTPUT' variable")
	}

	for key, value := range vars {
		placeholder := fmt.Sprintf("{vars.%s}", key)
		command = strings.ReplaceAll(command, placeholder, value)
	}
	return command, nil
}

var privateSshKeyStr string

// Start runs a scan
func (c Controller) Start(fleetName, command string, delete bool, input, outputPath1, chunksFolder string, module *models.Module) {
	var isFolderOut bool
	start := time.Now()
	privateSshKeyStr = c.Configs.SSHKeys.PrivateFile
	provider := c.Configs.Settings.Provider
	providerId := GetProvider(provider)
	if providerId == 0 {
		utils.Log.Fatal(models.ErrNotAvailableCustomVps)
	}

	port := c.Configs.Providers[provider].Port
	username := c.Configs.Providers[provider].Username

	input, inputOk := module.Vars["INPUT"]
	outputPath, outputOk := module.Vars["OUTPUT"]
	if !inputOk || !outputOk {
		utils.Log.Fatal("INPUT and OUTPUT vars are required in module")
	}

	timeStamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	// TODO: use a proper temp folder function so that it can run on windows too
	tempFolder := filepath.Join("/tmp", "fleex-"+"1698879444435075000")

	if chunksFolder != "" {
		tempFolder = chunksFolder
	}

	// Make local temp folder
	tempFolderInput := filepath.Join(tempFolder, "input")
	tempFolderFiles := filepath.Join(tempFolder, "files")
	utils.MakeFolder(tempFolder)
	utils.MakeFolder(tempFolderInput)
	utils.MakeFolder(tempFolderFiles)
	utils.Log.Info("Scan started!")

	// Input file to string

	fleet := c.GetFleet(fleetName)
	if len(fleet) < 1 {
		utils.Log.Fatal("No fleet found")
	}

	// Send additional vars files (excluding "INPUT" and "OUTPUT") via SCP
	for key, value := range module.Vars {
		if key != "INPUT" && key != "OUTPUT" && isFile(value) {
			newFileName := "/tmp/fleex-" + timeStamp + "-chunk-file-" + value
			module.Vars[key] = newFileName
			sendFileToFleet(value, newFileName, fleet, port, username, privateSshKeyStr)
		}
	}

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

	fleetNames := make(chan *p.Box, len(fleet))
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

				conn, err := sshutils.Connect(l.IP+":"+strconv.Itoa(port), username, privateSshKeyStr)
				if err != nil {
					utils.Log.Fatal(err)
				}
				// Send input file via SCP
				err = scp.NewSCP(conn.Client).SendFile(filepath.Join(tempFolderInput, "chunk-"+boxName), "/tmp/fleex-"+timeStamp+"-chunk-"+boxName)
				if err != nil {
					utils.Log.Fatal("Failed to send file: ", err)
				}

				chunkInputFile := "/tmp/fleex-" + timeStamp + "-chunk-" + boxName
				chunkOutputFile := "/tmp/fleex-" + timeStamp + "-chunk-out-" + boxName

				// Replace labels and craft final command
				module.Vars["INPUT"] = chunkInputFile
				module.Vars["OUTPUT"] = chunkOutputFile
				finalCommand, err := ReplaceCommandVars(command, module.Vars)
				if err != nil {
					utils.Log.Fatal(err)
				}

				sshutils.RunCommand(finalCommand, l.IP, port, username, privateSshKeyStr)

				err = scp.NewSCP(conn.Client).ReceiveFile(chunkOutputFile, filepath.Join(tempFolder, "chunk-out-"+boxName))
				if err != nil {
					os.Remove(filepath.Join(tempFolder, "chunk-out-"+boxName))
					err := scp.NewSCP(conn.Client).ReceiveDir(chunkOutputFile, filepath.Join(tempFolder, "chunk-out-"+boxName), nil)
					if err != nil {
						utils.Log.Fatal("SEND DIR ERROR: ", err)
					}
				}

				// Remove input chunk file from remote box to save space
				sshutils.RunCommand("sudo rm -rf "+chunkInputFile+" "+chunkOutputFile, l.IP, port, username, privateSshKeyStr)

				if delete {
					// TODO: Not the best way to delete a box, if this program crashes/is stopped
					// before reaching this line the box won't be deleted. It's better to setup
					// a cron/command on the box directly.
					c.DeleteBoxByID(l.ID, "", providerId)
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

	// TODO: Get rid of bash and do this using Go

	if isFolderOut {
		SaveInFolder(tempFolder, outputPath)
	} else {
		utils.RunCommand("cat "+filepath.Join(tempFolder, "chunk-out-*")+" > "+outputPath, true)
	}

	if chunksFolder == "" {
		//utils.RunCommand("rm -rf " + filepath.Join(tempFolder, "chunk-out-*"))
		os.RemoveAll(tempFolder)
	}

}

func SaveInFolder(inputPath string, outputPath string) {
	utils.MakeFolder(outputPath)

	filepath.Walk(inputPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				if !strings.Contains(info.Name(), "chunk-") {
					utils.RunCommand("cp "+path+" "+outputPath, true)
				}
			}
			return nil
		})
}

func IsDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), err
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func sendFileToFleet(filePath, destinationPath string, fleet []p.Box, port int, username, privateKey string) error {
	for _, box := range fleet {
		conn, err := sshutils.Connect(box.IP+":"+strconv.Itoa(port), username, privateKey)
		if err != nil {
			return err
		}

		err = scp.NewSCP(conn.Client).SendFile(filePath, destinationPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c Controller) VerticalStart(fleetName, command string, delete bool, outputPath1, chunksFolder string, module *models.Module, splitVar string) {
	var isFolderOut bool
	start := time.Now()
	privateSshKeyStr = c.Configs.SSHKeys.PrivateFile
	provider := c.Configs.Settings.Provider
	providerId := GetProvider(provider)
	if providerId == 0 {
		utils.Log.Fatal(models.ErrNotAvailableCustomVps)
	}

	port := c.Configs.Providers[provider].Port
	username := c.Configs.Providers[provider].Username

	outputPath, outputOk := module.Vars["OUTPUT"]
	if !outputOk {
		utils.Log.Fatal("OUTPUT var is required in module")
	}

	splitFilePath, splitOk := module.Vars[splitVar]
	if !splitOk {
		utils.Log.Fatalf("Variable '%s' not found in params", splitVar)
	}

	if !isFile(splitFilePath) {
		utils.Log.Fatalf("File '%s' specified in variable '%s' does not exist", splitFilePath, splitVar)
	}

	timeStamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	tempFolder := filepath.Join("/tmp", "fleex-"+timeStamp)

	if chunksFolder != "" {
		tempFolder = chunksFolder
	}

	tempFolderInput := filepath.Join(tempFolder, "input")
	tempFolderFiles := filepath.Join(tempFolder, "files")
	utils.MakeFolder(tempFolder)
	utils.MakeFolder(tempFolderInput)
	utils.MakeFolder(tempFolderFiles)
	utils.Log.Info("Vertical scan started!")

	fleet := c.GetFleet(fleetName)
	if len(fleet) < 1 {
		utils.Log.Fatal("No fleet found")
	}

	for key, value := range module.Vars {
		if key != splitVar && key != "OUTPUT" && isFile(value) {
			newFileName := "/tmp/fleex-" + timeStamp + "-chunk-file-" + filepath.Base(value)
			module.Vars[key] = newFileName
			sendFileToFleet(value, newFileName, fleet, port, username, privateSshKeyStr)
		}
	}

	file, err := os.Open(splitFilePath)
	if err != nil {
		utils.Log.Fatal(err)
	}

	linesCount, err := lineCounter(file)
	if err != nil {
		utils.Log.Fatal(err)
	}

	utils.Log.Debug("Fleet count: ", len(fleet))
	utils.Log.Debug("Total lines to split: ", linesCount)
	linesPerChunk := linesCount / len(fleet)
	linesPerChunkRest := linesCount % len(fleet)

	names := make(chan string)
	readerr := make(chan error)

	go GetLine(splitFilePath, names, readerr)
	counter := 1
	asd := []string{}

	x := 1

loop:
	for {
		select {
		case name := <-names:
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

	utils.Log.Debug("Generated file chunks for split variable")

	fleetNames := make(chan *p.Box, len(fleet))
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

				conn, err := sshutils.Connect(l.IP+":"+strconv.Itoa(port), username, privateSshKeyStr)
				if err != nil {
					utils.Log.Fatal(err)
				}

				chunkPath := filepath.Join(tempFolderInput, "chunk-"+boxName)
				if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
					utils.Log.Debug("No chunk for box ", boxName, ", skipping")
					processGroup.Done()
					continue
				}

				remoteSplitFile := "/tmp/fleex-" + timeStamp + "-chunk-" + boxName
				err = scp.NewSCP(conn.Client).SendFile(chunkPath, remoteSplitFile)
				if err != nil {
					utils.Log.Fatal("Failed to send file: ", err)
				}

				chunkOutputFile := "/tmp/fleex-" + timeStamp + "-chunk-out-" + boxName

				localVars := make(map[string]string)
				for k, v := range module.Vars {
					localVars[k] = v
				}
				localVars[splitVar] = remoteSplitFile
				localVars["OUTPUT"] = chunkOutputFile

				finalCommand, err := ReplaceVerticalCommandVars(command, localVars)
				if err != nil {
					utils.Log.Fatal(err)
				}

				sshutils.RunCommand(finalCommand, l.IP, port, username, privateSshKeyStr)

				err = scp.NewSCP(conn.Client).ReceiveFile(chunkOutputFile, filepath.Join(tempFolder, "chunk-out-"+boxName))
				if err != nil {
					os.Remove(filepath.Join(tempFolder, "chunk-out-"+boxName))
					err := scp.NewSCP(conn.Client).ReceiveDir(chunkOutputFile, filepath.Join(tempFolder, "chunk-out-"+boxName), nil)
					if err != nil {
						utils.Log.Fatal("SEND DIR ERROR: ", err)
					}
				}

				sshutils.RunCommand("sudo rm -rf "+remoteSplitFile+" "+chunkOutputFile, l.IP, port, username, privateSshKeyStr)

				if delete {
					c.DeleteBoxByID(l.ID, "", providerId)
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

	duration := time.Since(start)
	utils.Log.Info("Vertical scan done! Took ", duration, ". Output file: ", outputPath)

	if isFolderOut {
		SaveInFolder(tempFolder, outputPath)
	} else {
		utils.RunCommand("cat "+filepath.Join(tempFolder, "chunk-out-*")+" > "+outputPath, true)
	}

	if chunksFolder == "" {
		os.RemoveAll(tempFolder)
	}
}
