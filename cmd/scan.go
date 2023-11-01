package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Send a command to a fleet, but also with files upload & chunks splitting",
	Run: func(cmd *cobra.Command, args []string) {
		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		paramsFlag, _ := cmd.Flags().GetStringSlice("params")
		commandFlag, _ := cmd.Flags().GetString("command")
		deleteFlag, _ := cmd.Flags().GetBool("delete")
		fleetNameFlag, _ := cmd.Flags().GetString("name")
		inputFlag, _ := cmd.Flags().GetString("input")
		output, _ := cmd.Flags().GetString("output")
		portFlag, _ := cmd.Flags().GetInt("port")
		usernameFlag, _ := cmd.Flags().GetString("username")
		passwordFlag, _ := cmd.Flags().GetString("password")
		templatePathFlag, _ := cmd.Flags().GetString("template")

		chunksFolder, _ := cmd.Flags().GetString("chunks-folder")
		if globalConfig.Settings.Provider != providerFlag && providerFlag == "" {
			providerFlag = globalConfig.Settings.Provider
		}
		provider := controller.GetProvider(providerFlag)
		if provider == -1 {
			utils.Log.Fatal(models.ErrInvalidProvider)
		}

		providerInfo := globalConfig.Providers[providerFlag]
		if portFlag != -1 {
			providerInfo.Port = portFlag
		}
		if usernameFlag != "" {
			providerInfo.Username = usernameFlag
		}
		if passwordFlag != "" {
			providerInfo.Password = passwordFlag
		}

		module := &models.Module{}
		module.Vars = make(map[string]string)

		if templatePathFlag != "" {
			var err error
			module, err = readYAMLConfig(templatePathFlag)
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
			// command := replaceCommandVars(commandFlag, config.Vars)
			module.Commands = []string{commandFlag}
		} else if len(module.Commands) == 0 {
			log.Fatal("No commands specified.")
		}
		// else {
		// 	for i, command := range config.Commands {
		// 		config.Commands[i] = replaceCommandVars(command, config.Vars)
		// 	}
		// }

		finalCommand := ""
		if len(module.Commands) > 0 {
			finalCommand = module.Commands[0]
		}

		log.Fatal(1, module.Vars, " ", finalCommand)

		newController := controller.NewController(globalConfig)
		newController.Start(fleetNameFlag, finalCommand, deleteFlag, inputFlag, output, chunksFolder)

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
	scanCmd.Flags().StringP("template", "", "", "Specify path to a YAML template file")
	// scanCmd.Flags().StringP("module", "m", "", "Scan modules")

	// scanCmd.MarkFlagRequired("output")
	// scanCmd.MarkFlagRequired("command")
}

func replaceCommandVars(command string, vars map[string]string) string {
	for key, value := range vars {
		placeholder := fmt.Sprintf("{vars.%s}", key)
		command = strings.ReplaceAll(command, placeholder, value)
	}
	return command
}

func readYAMLConfig(path string) (*models.Module, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &models.Module{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
