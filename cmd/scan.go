package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Send a command to a fleet, but also with files upload & chunks splitting",
	Long: `Run scans on a fleet. Supports three modes:

1. Horizontal scan (default):
   fleex scan -n myfleet -c "nuclei -l {INPUT} -o {OUTPUT}" -i targets.txt -o results.txt
   Splits the input file (targets) across fleet boxes, each box uses the same command.

2. Vertical scan:
   fleex scan -n myfleet --vertical --split-var WORDLIST -c "puredns bruteforce {vars.WORDLIST} {vars.TARGET} -o {OUTPUT}" -p TARGET:tesla.com -p WORDLIST:wordlist.txt -o results.txt
   Splits the wordlist across fleet boxes, all boxes target the same host.

3. Workflow mode (multi-step pipeline):
   fleex scan -n myfleet --workflow full-recon -i targets.txt -o results.txt

In workflow mode, each machine:
  1. Takes 1 chunk of the input
  2. Executes ALL steps in sequence
  3. Returns the final output`,
	Run: func(cmd *cobra.Command, args []string) {
		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		paramsFlag, _ := cmd.Flags().GetStringSlice("params")
		commandFlag, _ := cmd.Flags().GetString("command")
		deleteFlag, _ := cmd.Flags().GetBool("delete")
		fleetNameFlag, _ := cmd.Flags().GetString("name")
		inputFlag, _ := cmd.Flags().GetString("input")
		output, _ := cmd.Flags().GetString("output")
		modulePathFlag, _ := cmd.Flags().GetString("module")
		chunksFolder, _ := cmd.Flags().GetString("chunks-folder")

		workflowName, _ := cmd.Flags().GetString("workflow")
		workflowFile, _ := cmd.Flags().GetString("workflow-file")

		verticalFlag, _ := cmd.Flags().GetBool("vertical")
		splitVarFlag, _ := cmd.Flags().GetString("split-var")

		if workflowName != "" || workflowFile != "" {
			runWorkflowMode(cmd, fleetNameFlag, inputFlag, output, chunksFolder, deleteFlag, workflowName, workflowFile)
			return
		}

		module := &models.Module{}
		module.Vars = make(map[string]string)

		if modulePathFlag != "" {
			var err error
			module, err = utils.ReadModuleFile(modulePathFlag)
			if err != nil {
				log.Fatalf("Error reading YAML file: %v", err)
			}
		}

		for _, param := range paramsFlag {
			splits := strings.SplitN(param, ":", 2)
			if len(splits) == 2 {
				key, value := splits[0], splits[1]
				module.Vars[key] = value
			}
		}

		if commandFlag != "" {
			module.Commands = []string{commandFlag}
		} else if len(module.Commands) == 0 {
			log.Fatal("No commands specified.")
		}

		finalCommand := ""
		if len(module.Commands) > 0 {
			finalCommand = module.Commands[0]
		}

		newController := controller.NewController(globalConfig)

		if verticalFlag {
			if splitVarFlag == "" {
				log.Fatal("Vertical scan requires --split-var to specify which variable to split (e.g., WORDLIST)")
			}
			if _, ok := module.Vars[splitVarFlag]; !ok {
				log.Fatalf("Variable '%s' not found in params. Use -p %s:/path/to/file", splitVarFlag, splitVarFlag)
			}
			newController.VerticalStart(fleetNameFlag, finalCommand, deleteFlag, output, chunksFolder, module, splitVarFlag)
		} else {
			newController.Start(fleetNameFlag, finalCommand, deleteFlag, inputFlag, output, chunksFolder, module)
		}
	},
}

func runWorkflowMode(cmd *cobra.Command, fleetName, input, output, chunksFolder string, deleteFleet bool, workflowName, workflowFile string) {
	var workflow *models.Workflow
	var err error

	if workflowFile != "" {
		workflow, err = utils.ReadWorkflowFile(workflowFile)
	} else {
		workflow, err = utils.ReadWorkflowFile(workflowName)
	}

	if err != nil {
		utils.Log.Fatal("Failed to load workflow: ", err)
	}

	paramsFlag, _ := cmd.Flags().GetStringSlice("params")
	if workflow.Vars == nil {
		workflow.Vars = make(map[string]string)
	}
	for _, param := range paramsFlag {
		splits := strings.SplitN(param, ":", 2)
		if len(splits) == 2 {
			workflow.Vars[splits[0]] = splits[1]
		}
	}

	newController := controller.NewController(globalConfig)

	fleet := newController.GetFleet(fleetName)
	if len(fleet) == 0 {
		utils.Log.Fatal("Fleet not found: ", fleetName)
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")

	opts := models.WorkflowOptions{
		Workflow:     workflow,
		FleetName:    fleetName,
		Input:        input,
		Output:       output,
		ChunksFolder: chunksFolder,
		Delete:       deleteFleet,
		DryRun:       dryRun,
		Verbose:      verbose,
	}

	results, err := newController.RunWorkflow(opts)
	if err != nil {
		utils.Log.Fatal(err)
	}

	if !dryRun {
		successCount := 0
		for _, r := range results {
			if r.Success {
				successCount++
			}
		}
		fmt.Printf("\nWorkflow complete: %d/%d successful\n", successCount, len(results))
	}
}

var scanListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available workflows",
	Run: func(cmd *cobra.Command, args []string) {
		workflows, err := utils.ListWorkflows()
		if err != nil {
			utils.Log.Fatal(err)
		}

		if len(workflows) == 0 {
			fmt.Println("No workflows found. Run 'fleex init' to create default workflows.")
			return
		}

		fmt.Println("\n=== AVAILABLE WORKFLOWS ===\n")
		fmt.Printf("%-25s %-50s %-10s\n", "NAME", "DESCRIPTION", "STEPS")
		fmt.Printf("%-25s %-50s %-10s\n", strings.Repeat("-", 25), strings.Repeat("-", 50), strings.Repeat("-", 10))

		for _, name := range workflows {
			workflow, err := utils.ReadWorkflowFile(name)
			if err != nil {
				continue
			}
			fmt.Printf("%-25s %-50s %-10d\n", name, workflow.Description, len(workflow.Steps))
		}
		fmt.Println()
	},
}

var scanShowCmd = &cobra.Command{
	Use:   "show [workflow-name]",
	Short: "Show workflow details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workflowName := args[0]

		workflow, err := utils.ReadWorkflowFile(workflowName)
		if err != nil {
			utils.Log.Fatal("Workflow not found: ", workflowName)
		}

		fmt.Printf("\n=== %s ===\n\n", strings.ToUpper(workflow.Name))
		fmt.Printf("Description: %s\n", workflow.Description)
		if workflow.Author != "" {
			fmt.Printf("Author:      %s\n", workflow.Author)
		}

		scaleMode := workflow.ScaleMode
		if scaleMode == "" {
			scaleMode = "horizontal"
		}
		fmt.Printf("Scale mode:  %s\n", scaleMode)
		if workflow.SplitVar != "" {
			fmt.Printf("Split var:   %s\n", workflow.SplitVar)
		}

		if len(workflow.Vars) > 0 {
			fmt.Println("\nVariables:")
			for k, v := range workflow.Vars {
				if v == "" {
					fmt.Printf("  %s: (required)\n", k)
				} else {
					fmt.Printf("  %s: %s\n", k, v)
				}
			}
		}

		if len(workflow.Setup) > 0 {
			fmt.Println("\nSetup (runs on all boxes first):")
			for _, c := range workflow.Setup {
				fmt.Printf("  $ %s\n", c)
			}
		}

		fmt.Println("\nSteps (run sequentially on each chunk):")
		for i, step := range workflow.Steps {
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

			if len(step.Command) > 80 {
				fmt.Printf("     $ %s...\n", step.Command[:80])
			} else {
				fmt.Printf("     $ %s\n", step.Command)
			}
			if step.Timeout != "" {
				fmt.Printf("     timeout: %s\n", step.Timeout)
			}
		}

		if workflow.Output.Aggregate != "" || workflow.Output.Deduplicate {
			fmt.Println("\nOutput:")
			if workflow.Output.Aggregate != "" {
				fmt.Printf("  aggregate: %s\n", workflow.Output.Aggregate)
			}
			if workflow.Output.Deduplicate {
				fmt.Println("  deduplicate: true")
			}
		}
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.AddCommand(scanListCmd)
	scanCmd.AddCommand(scanShowCmd)

	scanCmd.Flags().StringSliceP("params", "", []string{}, "Set parameters in the format KEY:VALUE")
	scanCmd.Flags().StringP("name", "n", "pwn", "Fleet name")
	scanCmd.Flags().StringP("command", "c", "", "Command to send. Supports {{INPUT}} and {{OUTPUT}}")
	scanCmd.Flags().StringP("input", "i", "", "Input file")
	scanCmd.Flags().StringP("output", "o", "", "Output file path. Made from concatenating all output chunks from all boxes")
	scanCmd.Flags().StringP("chunks-folder", "", "", "Output folder containing output chunks. If empty it will use /tmp/<unix_timestamp>")
	scanCmd.Flags().StringP("provider", "p", "", "VPS provider (Supported: linode, digitalocean, vultr)")
	scanCmd.Flags().IntP("port", "", -1, "SSH port")
	scanCmd.Flags().StringP("username", "U", "", "SSH username")
	scanCmd.Flags().StringP("password", "P", "", "SSH password")
	scanCmd.Flags().BoolP("delete", "d", false, "Delete boxes as soon as they finish their job")
	scanCmd.Flags().StringP("module", "", "", "Specify path to a YAML module file")

	scanCmd.Flags().StringP("workflow", "w", "", "Workflow name (multi-step mode)")
	scanCmd.Flags().StringP("workflow-file", "", "", "Custom workflow file path")
	scanCmd.Flags().BoolP("dry-run", "", false, "Show what would be executed (workflow mode)")
	scanCmd.Flags().BoolP("verbose", "v", false, "Show detailed output (workflow mode)")

	scanCmd.Flags().BoolP("vertical", "", false, "Enable vertical scanning (split wordlist instead of targets)")
	scanCmd.Flags().StringP("split-var", "", "", "Variable name to split in vertical mode (e.g., WORDLIST)")
}
