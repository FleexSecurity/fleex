package cmd

import (
	"log"
	"strings"

	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Send a command to a fleet, but also with files upload & chunks splitting",
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
		newController.Start(fleetNameFlag, finalCommand, deleteFlag, inputFlag, output, chunksFolder, module)

	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
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

	// scanCmd.MarkFlagRequired("output")
	// scanCmd.MarkFlagRequired("command")
}
