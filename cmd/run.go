package cmd

import (
	"log"

	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Send a command to a fleet",
	Run: func(cmd *cobra.Command, args []string) {
		var token string

		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		fleetName, _ := cmd.Flags().GetString("name")
		command, _ := cmd.Flags().GetString("command")
		providerFlag, _ := cmd.Flags().GetString("provider")
		port, _ := cmd.Flags().GetInt("port")
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")

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

		controller.RunCommand(fleetName, command, token, port, username, password, provider)

	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringP("name", "n", "pwn", "Box name")
	runCmd.Flags().StringP("command", "c", "", "Command to send")
	runCmd.Flags().IntP("port", "", -1, "SSH port")
	runCmd.Flags().StringP("username", "U", "", "SSH username")
	runCmd.Flags().StringP("password", "P", "", "SSH password")
	runCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")

	runCmd.MarkFlagRequired("command")
}
