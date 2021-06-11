package scan

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/sw33tLie/fleex/pkg/controller"
	"github.com/sw33tLie/fleex/pkg/utils"
)

var log = logrus.New()

// Start runs a scan
func Start(fleetName string, command string, delete bool, input string, output string, token string, provider controller.Provider) {
	log.Info("Scan started. Input: ", input, " output: ", output)

	// Make local temp folder
	tempFolder := path.Join("/tmp", strconv.FormatInt(time.Now().UnixNano(), 10))

	// Create temp folder
	err := os.Mkdir(tempFolder, 0755)
	if err != nil {
		log.Fatal(err)
	}

	// Input file to string
	inputString := utils.FileToString(input)

	fleet := controller.GetFleet(fleetName, token, provider)

	linesCount := linesCount(inputString)
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
			chunkPath := path.Join(tempFolder, "chunk-"+fleetName+"-"+strconv.Itoa(counter/linesPerChunk))
			utils.StringToFile(chunkPath, chunkContent)
			inputFiles = append(inputFiles, chunkPath)

			fmt.Println("" + chunkPath)

			chunkContent = ""
		}
		counter++
	}

	if scanner.Err() != nil {
		log.Println(scanner.Err())
	}
}
