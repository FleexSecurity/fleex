package cmd

import (
	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Send a command to a fleet",
	Run: func(cmd *cobra.Command, args []string) {
		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		fleetName, _ := cmd.Flags().GetString("name")
		commandFlag, _ := cmd.Flags().GetString("command")
		providerFlag, _ := cmd.Flags().GetString("provider")
		portFlag, _ := cmd.Flags().GetInt("port")
		usernameFlag, _ := cmd.Flags().GetString("username")
		passwordFlag, _ := cmd.Flags().GetString("password")

		if providerFlag != "" {
			globalConfig.Settings.Provider = providerFlag
		}
		providerFlag = globalConfig.Settings.Provider

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

		globalConfig.Providers[providerFlag] = providerInfo

		newController := controller.NewController(globalConfig)
		newController.RunCommand(fleetName, commandFlag)

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
