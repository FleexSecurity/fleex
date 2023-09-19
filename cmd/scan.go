package cmd

import (
	"io/ioutil"
	"log"

	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/scan"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type Module struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Author      string `yaml:"author"`
	Command     string `yaml:"command"`
}

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Send a command to a fleet, but also with files upload & chunks splitting",
	Run: func(cmd *cobra.Command, args []string) {
		var token string

		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		commandFlag, _ := cmd.Flags().GetString("command")
		deleteFlag, _ := cmd.Flags().GetBool("delete")
		fleetNameFlag, _ := cmd.Flags().GetString("name")
		inputFlag, _ := cmd.Flags().GetString("input")
		output, _ := cmd.Flags().GetString("output")
		moduleFlag, _ := cmd.Flags().GetString("module")
		port, _ := cmd.Flags().GetInt("port")
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")

		chunksFolder, _ := cmd.Flags().GetString("chunks-folder")
		if globalConfig.Settings.Provider != providerFlag && providerFlag == "" {
			providerFlag = globalConfig.Settings.Provider
		}
		provider := controller.GetProvider(providerFlag)
		if provider == -1 {
			log.Fatal("invalid provider")
		}

		if port == -1 {
			port = globalConfig.Providers[providerFlag].Port
		}
		if username == "" {
			username = globalConfig.Providers[providerFlag].Username
		}
		if password == "" {
			password = globalConfig.Providers[providerFlag].Password
		}
		token = globalConfig.Providers[providerFlag].Token

		var module Module

		if moduleFlag != "" {
			selectedModule := module.getModule(moduleFlag)
			commandFlag = selectedModule.Command
			utils.Log.Info(selectedModule.Name, ": ", selectedModule.Description)
			utils.Log.Info("Created by: ", selectedModule.Author)
		}

		if commandFlag == "" {
			utils.Log.Fatal("Command not found, insert a command or module")
		}

		scan.Start(fleetNameFlag, commandFlag, deleteFlag, inputFlag, output, chunksFolder, token, port, username, password, provider)

	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
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
	scanCmd.Flags().StringP("module", "m", "", "Scan modules")

	scanCmd.MarkFlagRequired("output")
	// scanCmd.MarkFlagRequired("command")
}

func (m *Module) getModule(modulename string) *Module {
	home, _ := homedir.Dir()
	yamlFile, err := ioutil.ReadFile(home + "/fleex/modules/" + modulename + ".yaml")
	if err != nil {
		utils.Log.Fatal("yamlFile.Get:", err)
	}
	err = yaml.Unmarshal(yamlFile, m)
	if err != nil {
		utils.Log.Fatal("Unmarshal:", err)
	}
	return m
}
