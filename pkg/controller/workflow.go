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

	scaleMode := opts.Workflow.ScaleMode
	if scaleMode == "" {
		scaleMode = "horizontal"
	}

	if scaleMode == "vertical" && opts.Workflow.SplitVar == "" {
		return nil, fmt.Errorf("vertical scale-mode requires split-var to be specified")
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

	if len(opts.Workflow.Files) > 0 {
		progress.StartFileTransfer()
		err := c.transferFilesToFleet(fleet, opts.Workflow.Files, opts.Workflow.Vars, port, username, privateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("file transfer failed: %w", err)
		}
		progress.FileTransferDone(len(opts.Workflow.Files))
	}

	var chunkFiles []string
	var err error
	splitVarChunksMap := make(map[string][]string)

	if scaleMode == "vertical" {
		splitVarFile, ok := opts.Workflow.Vars[opts.Workflow.SplitVar]
		if !ok {
			return nil, fmt.Errorf("split-var '%s' not found in workflow vars", opts.Workflow.SplitVar)
		}
		splitVarFile = utils.ExpandPath(splitVarFile)

		progress.StartChunking(splitVarFile)
		splitVarChunks, err := c.splitInputIntoChunks(splitVarFile, tempFolderInput, opts.FleetName+"-split-"+opts.Workflow.SplitVar, len(fleet))
		if err != nil {
			return nil, fmt.Errorf("failed to split %s: %w", opts.Workflow.SplitVar, err)
		}
		splitVarChunksMap[opts.Workflow.SplitVar] = splitVarChunks
		progress.ChunkingDone(len(splitVarChunks))

		chunkFiles = make([]string, len(fleet))
		for i := range chunkFiles {
			chunkFiles[i] = ""
		}
	} else {
		progress.StartChunking(opts.Input)
		chunkFiles, err = c.splitInputIntoChunks(opts.Input, tempFolderInput, opts.FleetName, len(fleet))
		if err != nil {
			return nil, fmt.Errorf("failed to split input: %w", err)
		}
		progress.ChunkingDone(len(chunkFiles))
	}

	for _, step := range opts.Workflow.Steps {
		if step.ScaleMode == "vertical" && step.SplitVar != "" {
			if _, exists := splitVarChunksMap[step.SplitVar]; !exists {
				splitVarFile, ok := opts.Workflow.Vars[step.SplitVar]
				if !ok {
					return nil, fmt.Errorf("step '%s' split-var '%s' not found in workflow vars", step.Name, step.SplitVar)
				}
				splitVarFile = utils.ExpandPath(splitVarFile)
				splitVarChunks, err := c.splitInputIntoChunks(splitVarFile, tempFolderInput, opts.FleetName+"-split-"+step.SplitVar, len(fleet))
				if err != nil {
					return nil, fmt.Errorf("failed to split %s for step %s: %w", step.SplitVar, step.Name, err)
				}
				splitVarChunksMap[step.SplitVar] = splitVarChunks
			}
		}
	}

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

		boxSplitVarChunks := make(map[string]string)
		for varName, chunks := range splitVarChunksMap {
			if i < len(chunks) {
				boxSplitVarChunks[varName] = chunks[i]
			}
		}

		fleetChan <- boxWithChunk{
			box:              &box,
			chunkFile:        chunkFile,
			splitVarChunks:   boxSplitVarChunks,
			index:            i,
			scaleMode:        scaleMode,
			splitVar:         opts.Workflow.SplitVar,
		}
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
	box              *provider.Box
	chunkFile        string
	splitVarChunks   map[string]string
	index            int
	scaleMode        string
	splitVar         string
}

func (c Controller) dryRunWorkflow(opts models.WorkflowOptions, fleet []provider.Box) ([]models.WorkflowResult, error) {
	ui.Info("Dry run mode - showing what would be executed:")
	fmt.Println()

	scaleMode := opts.Workflow.ScaleMode
	if scaleMode == "" {
		scaleMode = "horizontal"
	}

	fmt.Printf("Scale mode: %s\n", scaleMode)
	if scaleMode == "vertical" {
		fmt.Printf("Split variable: %s\n", opts.Workflow.SplitVar)
	}
	fmt.Println()

	if len(opts.Workflow.Setup) > 0 {
		fmt.Println("Setup commands (run on all boxes):")
		for _, cmd := range opts.Workflow.Setup {
			fmt.Printf("  $ %s\n", cmd)
		}
		fmt.Println()
	}

	if len(opts.Workflow.Files) > 0 {
		fmt.Println("Files to transfer (to all boxes):")
		for _, file := range opts.Workflow.Files {
			srcPath := utils.ExpandPath(file.Source)
			srcPath = utils.ReplaceWorkflowVars(srcPath, opts.Workflow.Vars)
			dstPath := utils.ReplaceWorkflowVars(file.Destination, opts.Workflow.Vars)
			fmt.Printf("  %s -> %s\n", srcPath, dstPath)
		}
		fmt.Println()
	}

	if scaleMode == "vertical" {
		splitVarFile := opts.Workflow.Vars[opts.Workflow.SplitVar]
		fmt.Printf("Split file: %s -> split into %d chunks\n\n", splitVarFile, len(fleet))
	} else {
		fmt.Printf("Input: %s -> split into %d chunks\n\n", opts.Input, len(fleet))
	}

	fmt.Println("Steps (run sequentially on each box):")
	for i, step := range opts.Workflow.Steps {
		stepHeader := fmt.Sprintf("%d. %s", i+1, step.Name)
		if step.Id != "" {
			stepHeader += fmt.Sprintf(" [id: %s]", step.Id)
		}
		fmt.Printf("  %s\n", stepHeader)

		stepScaleMode := step.ScaleMode
		if stepScaleMode == "" {
			if i == 0 {
				stepScaleMode = scaleMode
			} else {
				stepScaleMode = "local"
			}
		}
		fmt.Printf("     scale-mode: %s\n", stepScaleMode)
		if step.SplitVar != "" {
			fmt.Printf("     split-var: %s\n", step.SplitVar)
		}

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

	var currentInput string
	remoteSplitVarFiles := make(map[string]string)

	if item.scaleMode == "vertical" && item.splitVar != "" {
		if chunkPath, ok := item.splitVarChunks[item.splitVar]; ok && chunkPath != "" {
			remotePath := fmt.Sprintf("/tmp/fleex-%s-splitvar-%s-%s", timeStamp, item.splitVar, item.box.Label)
			err = scp.NewSCP(conn.Client).SendFile(chunkPath, remotePath)
			if err != nil {
				result.Error = fmt.Errorf("failed to send split-var chunk: %w", err)
				return result
			}
			remoteSplitVarFiles[item.splitVar] = remotePath
		}
		currentInput = ""
	} else {
		if item.chunkFile != "" {
			remoteChunkInput := fmt.Sprintf("/tmp/fleex-%s-chunk-%s", timeStamp, item.box.Label)
			err = scp.NewSCP(conn.Client).SendFile(item.chunkFile, remoteChunkInput)
			if err != nil {
				result.Error = fmt.Errorf("failed to send input chunk: %w", err)
				return result
			}
			currentInput = remoteChunkInput
		}
	}

	for varName, chunkPath := range item.splitVarChunks {
		if _, exists := remoteSplitVarFiles[varName]; !exists && chunkPath != "" {
			remotePath := fmt.Sprintf("/tmp/fleex-%s-splitvar-%s-%s", timeStamp, varName, item.box.Label)
			err = scp.NewSCP(conn.Client).SendFile(chunkPath, remotePath)
			if err != nil {
				result.Error = fmt.Errorf("failed to send split-var %s chunk: %w", varName, err)
				return result
			}
			remoteSplitVarFiles[varName] = remotePath
		}
	}

	var currentOutput string
	stepOutputs := make(map[string]string)

	for i, step := range opts.Workflow.Steps {
		if progress != nil {
			progress.UpdateStep(item.box.Label, step.Name, i+1)
		}

		currentOutput = fmt.Sprintf("/tmp/fleex-%s-step-%d-%s", timeStamp, i, item.box.Label)

		vars := make(map[string]string)
		for k, v := range opts.Workflow.Vars {
			vars[k] = v
		}

		stepScaleMode := step.ScaleMode
		if stepScaleMode == "" {
			if i == 0 {
				stepScaleMode = item.scaleMode
			} else {
				stepScaleMode = "local"
			}
		}

		if stepScaleMode == "vertical" {
			stepSplitVar := step.SplitVar
			if stepSplitVar == "" {
				stepSplitVar = item.splitVar
			}
			if remotePath, ok := remoteSplitVarFiles[stepSplitVar]; ok {
				vars[stepSplitVar] = remotePath
			}
		}

		if item.scaleMode == "vertical" && item.splitVar != "" {
			if remotePath, ok := remoteSplitVarFiles[item.splitVar]; ok {
				vars[item.splitVar] = remotePath
			}
		}

		if stepScaleMode == "local" && currentInput != "" {
			vars["INPUT"] = currentInput
		} else if stepScaleMode != "vertical" && currentInput != "" {
			vars["INPUT"] = currentInput
		}

		vars["OUTPUT"] = currentOutput

		command := step.Command
		for stepId, stepOutput := range stepOutputs {
			placeholder := fmt.Sprintf("{%s.OUTPUT}", stepId)
			command = strings.ReplaceAll(command, placeholder, stepOutput)
		}

		command = utils.ReplaceWorkflowVars(command, vars)

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

		if step.Id != "" {
			stepOutputs[step.Id] = currentOutput
		}

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

func (c Controller) transferFilesToFleet(fleet []provider.Box, files []models.FileTransfer, vars map[string]string, port int, username, privateKeyPath string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(fleet))

	for _, box := range fleet {
		wg.Add(1)
		go func(b provider.Box) {
			defer wg.Done()

			conn, err := sshutils.Connect(b.IP+":"+strconv.Itoa(port), username, privateKeyPath)
			if err != nil {
				errChan <- fmt.Errorf("[%s] connection failed: %w", b.Label, err)
				return
			}
			defer conn.Close()

			for _, file := range files {
				srcPath := utils.ExpandPath(file.Source)
				srcPath = utils.ReplaceWorkflowVars(srcPath, vars)
				dstPath := utils.ReplaceWorkflowVars(file.Destination, vars)

				err := scp.NewSCP(conn.Client).SendFile(srcPath, dstPath)
				if err != nil {
					errChan <- fmt.Errorf("[%s] failed to transfer %s: %w", b.Label, file.Source, err)
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
