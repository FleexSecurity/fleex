package controller

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/hnakamur/go-scp"

	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/provider"
	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/ui"
	"github.com/FleexSecurity/fleex/pkg/utils"
)

func (c Controller) BuildFleet(opts models.BuildOptions) ([]models.BuildResult, error) {
	fleet := c.GetFleet(opts.FleetName)
	if len(fleet) == 0 {
		return nil, fmt.Errorf("fleet %s not found", opts.FleetName)
	}

	providerName := c.Configs.Settings.Provider
	port := c.Configs.Providers[providerName].Port
	username := c.Configs.Providers[providerName].Username
	privateKeyPath := c.Configs.SSHKeys.PrivateFile

	results := make([]models.BuildResult, len(fleet))

	if opts.DryRun {
		ui.Info("Dry run mode - showing what would be executed:")
		for _, step := range opts.Recipe.Steps {
			fmt.Printf("  Step: %s\n", step.Name)
			for _, cmd := range step.Commands {
				cmdExpanded := utils.ReplaceBuildVars(cmd, opts.Recipe.Vars)
				fmt.Printf("    $ %s\n", cmdExpanded)
			}
		}
		return results, nil
	}

	progress := ui.NewBuildProgress(len(fleet))
	progress.Start(opts.Recipe.Name)

	fleetChan := make(chan *provider.Box, len(fleet))
	resultsChan := make(chan models.BuildResult, len(fleet))
	var wg sync.WaitGroup

	parallel := opts.Parallel
	if parallel <= 0 {
		parallel = 5
	}
	if parallel > len(fleet) {
		parallel = len(fleet)
	}

	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for box := range fleetChan {
				progress.StartBox(box.Label, len(opts.Recipe.Steps))
				result := c.buildBoxWithProgress(box, opts, port, username, privateKeyPath, progress)
				if result.Success {
					progress.BoxSuccess(box.Label)
				} else {
					errMsg := ""
					if result.Error != nil {
						errMsg = result.Error.Error()
					}
					progress.BoxFailed(box.Label, errMsg)
				}
				resultsChan <- result
			}
		}()
	}

	for i := range fleet {
		fleetChan <- &fleet[i]
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

	progress.Done()

	return results, nil
}

func (c Controller) buildBox(box *provider.Box, opts models.BuildOptions, port int, username, privateKeyPath string) models.BuildResult {
	return c.buildBoxWithProgress(box, opts, port, username, privateKeyPath, nil)
}

func (c Controller) waitForSSH(ip string, port int, username, privateKeyPath string, maxRetries int) (*sshutils.Connection, error) {
	var conn *sshutils.Connection
	var err error

	for i := 0; i < maxRetries; i++ {
		conn, err = sshutils.Connect(ip+":"+strconv.Itoa(port), username, privateKeyPath)
		if err == nil {
			return conn, nil
		}
		time.Sleep(5 * time.Second)
	}
	return nil, fmt.Errorf("SSH not available after %d attempts: %v", maxRetries, err)
}

func (c Controller) buildBoxWithProgress(box *provider.Box, opts models.BuildOptions, port int, username, privateKeyPath string, progress *ui.BuildProgress) models.BuildResult {
	result := models.BuildResult{
		BoxName: box.Label,
		Steps:   make([]models.StepResult, 0),
	}

	start := time.Now()

	if progress != nil {
		progress.UpdateStep(box.Label, "Waiting for SSH...", 0)
	}

	conn, err := c.waitForSSH(box.IP, port, username, privateKeyPath, 24)
	if err != nil {
		result.Error = err
		result.Duration = time.Since(start)
		return result
	}

	for _, file := range opts.Recipe.Files {
		srcPath := utils.ExpandPath(file.Source)
		srcPath = utils.ReplaceBuildVars(srcPath, opts.Recipe.Vars)
		dstPath := utils.ReplaceBuildVars(file.Destination, opts.Recipe.Vars)

		err := scp.NewSCP(conn.Client).SendFile(srcPath, dstPath)
		if err != nil {
			result.Error = fmt.Errorf("file transfer failed: %v", err)
			result.Duration = time.Since(start)
			return result
		}
	}

	for i, step := range opts.Recipe.Steps {
		if progress != nil {
			progress.UpdateStep(box.Label, step.Name, i+1)
		}

		stepResult := c.executeStep(box, step, opts, port, username, privateKeyPath)
		result.Steps = append(result.Steps, stepResult)

		if !stepResult.Success && !opts.ContinueErr && step.ContinueOn != "error" {
			result.Error = fmt.Errorf("step %s failed", step.Name)
			break
		}
	}

	if !opts.NoVerify && result.Error == nil {
		for _, verify := range opts.Recipe.Verify {
			verifyResult := c.runVerify(box, verify, opts, port, username, privateKeyPath)
			if !verifyResult {
				result.Error = fmt.Errorf("verification failed: %s", verify.Name)
				break
			}
		}
	}

	result.Duration = time.Since(start)
	result.Success = result.Error == nil
	return result
}

func (c Controller) executeStep(box *provider.Box, step models.BuildStep, opts models.BuildOptions, port int, username, privateKeyPath string) models.StepResult {
	result := models.StepResult{
		StepName: step.Name,
	}

	start := time.Now()
	maxRetries := step.Retries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		result.Retries = attempt
		allCommandsSuccess := true

		for _, cmd := range step.Commands {
			cmdExpanded := utils.ReplaceBuildVars(cmd, opts.Recipe.Vars)

			_, err := sshutils.RunCommandSilent(cmdExpanded, box.IP, port, username, privateKeyPath)
			if err != nil {
				allCommandsSuccess = false
				result.Output = fmt.Sprintf("command failed: %s - %v", cmdExpanded, err)
				break
			}
		}

		if allCommandsSuccess {
			result.Success = true
			result.Duration = time.Since(start)
			return result
		}

		if attempt < maxRetries-1 {
			time.Sleep(2 * time.Second)
		}
	}

	result.Success = false
	result.Duration = time.Since(start)
	return result
}

func (c Controller) runVerify(box *provider.Box, verify models.VerifyStep, opts models.BuildOptions, port int, username, privateKeyPath string) bool {
	_, err := sshutils.RunCommandSilent(verify.Command, box.IP, port, username, privateKeyPath)
	return err == nil
}

func (c Controller) VerifyFleet(opts models.BuildOptions) (map[string]bool, error) {
	fleet := c.GetFleet(opts.FleetName)
	if len(fleet) == 0 {
		return nil, fmt.Errorf("fleet %s not found", opts.FleetName)
	}

	providerName := c.Configs.Settings.Provider
	port := c.Configs.Providers[providerName].Port
	username := c.Configs.Providers[providerName].Username
	privateKeyPath := c.Configs.SSHKeys.PrivateFile

	results := make(map[string]bool)

	for _, box := range fleet {
		allPassed := true
		for _, verify := range opts.Recipe.Verify {
			passed := c.runVerify(&box, verify, opts, port, username, privateKeyPath)
			if !passed {
				allPassed = false
				utils.Log.Error("[", box.Label, "] Verification failed: ", verify.Name)
			} else if opts.Verbose {
				utils.Log.Info("[", box.Label, "] Verification passed: ", verify.Name)
			}
		}
		results[box.Label] = allPassed
	}

	return results, nil
}
