package controller

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hnakamur/go-scp"

	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/provider"
	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/ui"
	"github.com/FleexSecurity/fleex/pkg/utils"
)

func (c Controller) RunWorkflow(opts models.WorkflowOptions) ([]models.WorkflowResult, error) {
	start := time.Now()
	providerName := c.Configs.Settings.Provider
	port := c.Configs.Providers[providerName].Port
	username := c.Configs.Providers[providerName].Username
	privateKeyPath := c.Configs.SSHKeys.PrivateFile

	fleet := c.GetFleet(opts.FleetName)
	if len(fleet) == 0 {
		return nil, fmt.Errorf("fleet %s not found", opts.FleetName)
	}

	if opts.DryRun {
		return c.dryRunWorkflow(opts, fleet)
	}

	progress := ui.NewWorkflowProgress(len(fleet))
	progress.Start(opts.Workflow.Name, len(opts.Workflow.Steps))

	timeStamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	tempFolder := filepath.Join("/tmp", "fleex-workflow-"+timeStamp)
	tempFolderInput := filepath.Join(tempFolder, "input")
	tempFolderOutput := filepath.Join(tempFolder, "output")

	if opts.ChunksFolder != "" {
		tempFolder = opts.ChunksFolder
		tempFolderInput = filepath.Join(tempFolder, "input")
		tempFolderOutput = filepath.Join(tempFolder, "output")
	}

	utils.MakeFolder(tempFolder)
	utils.MakeFolder(tempFolderInput)
	utils.MakeFolder(tempFolderOutput)

	if len(opts.Workflow.Setup) > 0 {
		progress.StartSetup()
		err := c.runSetupCommands(fleet, opts.Workflow.Setup, port, username, privateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("setup failed: %w", err)
		}
		progress.SetupDone()
	}

	progress.StartChunking(opts.Input)
	chunkFiles, err := c.splitInputIntoChunks(opts.Input, tempFolderInput, opts.FleetName, len(fleet))
	if err != nil {
		return nil, fmt.Errorf("failed to split input: %w", err)
	}
	progress.ChunkingDone(len(chunkFiles))

	results := make([]models.WorkflowResult, len(fleet))
	resultsChan := make(chan models.WorkflowResult, len(fleet))
	fleetChan := make(chan boxWithChunk, len(fleet))
	var wg sync.WaitGroup

	for i := 0; i < len(fleet); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range fleetChan {
				progress.StartBox(item.box.Label, len(opts.Workflow.Steps))
				result := c.runWorkflowOnBox(item, opts, port, username, privateKeyPath, tempFolder, timeStamp, progress)
				if result.Success {
					progress.BoxSuccess(item.box.Label)
				} else {
					errMsg := ""
					if result.Error != nil {
						errMsg = result.Error.Error()
					}
					progress.BoxFailed(item.box.Label, errMsg)
				}
				resultsChan <- result
			}
		}()
	}

	for i, box := range fleet {
		chunkFile := ""
		if i < len(chunkFiles) {
			chunkFile = chunkFiles[i]
		}
		fleetChan <- boxWithChunk{box: &box, chunkFile: chunkFile, index: i}
	}
	close(fleetChan)

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	idx := 0
	for result := range resultsChan {
		results[idx] = result
		idx++
	}

	progress.StartAggregating()
	err = c.aggregateResults(tempFolderOutput, opts.Output, opts.Workflow.Output)
	if err != nil {
		return results, fmt.Errorf("aggregation failed: %w", err)
	}
	progress.AggregatingDone(opts.Output)

	if opts.Delete {
		for _, box := range fleet {
			providerId := GetProvider(providerName)
			c.DeleteBoxByID(box.ID, "", providerId)
		}
	}

	if opts.ChunksFolder == "" {
		os.RemoveAll(tempFolder)
	}

	progress.Done()
	utils.Log.Info("Workflow completed in ", time.Since(start))

	return results, nil
}

type boxWithChunk struct {
	box       *provider.Box
	chunkFile string
	index     int
}

func (c Controller) dryRunWorkflow(opts models.WorkflowOptions, fleet []provider.Box) ([]models.WorkflowResult, error) {
	ui.Info("Dry run mode - showing what would be executed:")
	fmt.Println()

	if len(opts.Workflow.Setup) > 0 {
		fmt.Println("Setup commands (run on all boxes):")
		for _, cmd := range opts.Workflow.Setup {
			fmt.Printf("  $ %s\n", cmd)
		}
		fmt.Println()
	}

	fmt.Printf("Input: %s -> split into %d chunks\n\n", opts.Input, len(fleet))

	fmt.Println("Steps (run sequentially on each box):")
	for i, step := range opts.Workflow.Steps {
		fmt.Printf("  %d. %s\n", i+1, step.Name)
		cmdExpanded := utils.ReplaceWorkflowVars(step.Command, opts.Workflow.Vars)
		fmt.Printf("     $ %s\n", cmdExpanded)
		if step.Timeout != "" {
			fmt.Printf("     timeout: %s\n", step.Timeout)
		}
	}
	fmt.Println()

	fmt.Printf("Output: %s\n", opts.Output)
	if opts.Workflow.Output.Aggregate != "" {
		fmt.Printf("  aggregate: %s\n", opts.Workflow.Output.Aggregate)
	}
	if opts.Workflow.Output.Deduplicate {
		fmt.Printf("  deduplicate: true\n")
	}

	return []models.WorkflowResult{}, nil
}

func (c Controller) runSetupCommands(fleet []provider.Box, commands []string, port int, username, privateKeyPath string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(fleet))

	for _, box := range fleet {
		wg.Add(1)
		go func(b provider.Box) {
			defer wg.Done()
			for _, cmd := range commands {
				output, err := sshutils.RunCommandWithOutput(cmd, b.IP, port, username, privateKeyPath)
				if err != nil {
					outputStr := strings.TrimSpace(string(output))
					if outputStr != "" {
						errChan <- fmt.Errorf("[%s] setup command failed: %s\n%s", b.Label, cmd, outputStr)
					} else {
						errChan <- fmt.Errorf("[%s] setup command failed: %s - %v", b.Label, cmd, err)
					}
					return
				}
			}
		}(box)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		return err
	}
	return nil
}

func (c Controller) splitInputIntoChunks(inputFile, outputDir, fleetName string, numChunks int) ([]string, error) {
	file, err := os.Open(inputFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(lines) == 0 {
		return nil, fmt.Errorf("input file is empty")
	}

	linesPerChunk := len(lines) / numChunks
	remainder := len(lines) % numChunks

	var chunkFiles []string
	lineIndex := 0

	for i := 0; i < numChunks; i++ {
		chunkSize := linesPerChunk
		if i < remainder {
			chunkSize++
		}

		if lineIndex >= len(lines) {
			break
		}

		endIndex := lineIndex + chunkSize
		if endIndex > len(lines) {
			endIndex = len(lines)
		}

		chunkFileName := filepath.Join(outputDir, fmt.Sprintf("chunk-%s-%d", fleetName, i+1))
		chunkContent := strings.Join(lines[lineIndex:endIndex], "\n") + "\n"

		if err := os.WriteFile(chunkFileName, []byte(chunkContent), 0644); err != nil {
			return nil, err
		}

		chunkFiles = append(chunkFiles, chunkFileName)
		lineIndex = endIndex
	}

	return chunkFiles, nil
}

func (c Controller) runWorkflowOnBox(item boxWithChunk, opts models.WorkflowOptions, port int, username, privateKeyPath, tempFolder, timeStamp string, progress *ui.WorkflowProgress) models.WorkflowResult {
	result := models.WorkflowResult{
		BoxName:     item.box.Label,
		StepResults: make([]models.WorkflowStepResult, 0),
	}

	conn, err := sshutils.Connect(item.box.IP+":"+strconv.Itoa(port), username, privateKeyPath)
	if err != nil {
		result.Error = fmt.Errorf("SSH connection failed: %w", err)
		return result
	}
	defer conn.Close()

	remoteChunkInput := fmt.Sprintf("/tmp/fleex-%s-chunk-%s", timeStamp, item.box.Label)
	err = scp.NewSCP(conn.Client).SendFile(item.chunkFile, remoteChunkInput)
	if err != nil {
		result.Error = fmt.Errorf("failed to send input chunk: %w", err)
		return result
	}

	currentInput := remoteChunkInput
	var currentOutput string

	for i, step := range opts.Workflow.Steps {
		if progress != nil {
			progress.UpdateStep(item.box.Label, step.Name, i+1)
		}

		currentOutput = fmt.Sprintf("/tmp/fleex-%s-step-%d-%s", timeStamp, i, item.box.Label)

		vars := make(map[string]string)
		for k, v := range opts.Workflow.Vars {
			vars[k] = v
		}
		vars["INPUT"] = currentInput
		vars["OUTPUT"] = currentOutput

		command := utils.ReplaceWorkflowVars(step.Command, vars)

		stepResult := models.WorkflowStepResult{
			StepName: step.Name,
		}

		output, err := sshutils.RunCommandWithOutput(command, item.box.IP, port, username, privateKeyPath)
		if err != nil {
			stepResult.Success = false
			outputStr := strings.TrimSpace(string(output))
			if outputStr != "" {
				stepResult.Output = outputStr
				result.Error = fmt.Errorf("step %s failed: %v\n%s", step.Name, err, outputStr)
			} else {
				stepResult.Output = err.Error()
				result.Error = fmt.Errorf("step %s failed: %w", step.Name, err)
			}
			result.StepResults = append(result.StepResults, stepResult)
			return result
		}

		stepResult.Success = true
		result.StepResults = append(result.StepResults, stepResult)

		currentInput = currentOutput
	}

	localOutputFile := filepath.Join(tempFolder, "output", fmt.Sprintf("output-%s", item.box.Label))
	err = scp.NewSCP(conn.Client).ReceiveFile(currentOutput, localOutputFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to receive output: %w", err)
		return result
	}

	cleanupCmd := fmt.Sprintf("rm -f /tmp/fleex-%s-*", timeStamp)
	sshutils.RunCommandSilent(cleanupCmd, item.box.IP, port, username, privateKeyPath)

	result.Success = true
	return result
}

func (c Controller) aggregateResults(outputDir, finalOutput string, outputConfig models.WorkflowOutput) error {
	files, err := os.ReadDir(outputDir)
	if err != nil {
		return err
	}

	var outputFiles []string
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "output-") {
			outputFiles = append(outputFiles, filepath.Join(outputDir, f.Name()))
		}
	}

	if len(outputFiles) == 0 {
		return fmt.Errorf("no output files found")
	}

	aggregate := outputConfig.Aggregate
	if aggregate == "" {
		aggregate = "concat"
	}

	var allLines []string

	for _, file := range outputFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		allLines = append(allLines, lines...)
	}

	switch aggregate {
	case "sort-unique":
		sort.Strings(allLines)
		allLines = uniqueStrings(allLines)
	case "concat":
		if outputConfig.Deduplicate {
			allLines = uniqueStrings(allLines)
		}
	}

	output := strings.Join(allLines, "\n")
	if len(output) > 0 && !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return os.WriteFile(finalOutput, []byte(output), 0644)
}

func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range input {
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
