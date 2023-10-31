package cmd

import (
	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Send a command to a fleet",
	Run: func(cmd *cobra.Command, args []string) {
		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		fleetName, _ := cmd.Flags().GetString("name")
		commandFlag, _ := cmd.Flags().GetString("command")
		portFlag, _ := cmd.Flags().GetInt("port")
		usernameFlag, _ := cmd.Flags().GetString("username")

		if providerFlag != "" {
			globalConfig.Settings.Provider = providerFlag
		}
		providerFlag = globalConfig.Settings.Provider

		vmInfo := models.GetVMInfo(providerFlag, fleetName, globalConfig)
		if vmInfo == nil {
			utils.Log.Fatal("Provider or custom VM not found")
		}

		if portFlag != -1 {
			vmInfo.Port = portFlag
		}
		if usernameFlag != "" {
			vmInfo.Username = usernameFlag
		}

		newController := controller.NewController(globalConfig)

		fleets := newController.GetFleet(fleetName)

		if len(fleets) == 0 {
			utils.Log.Fatal("Fleet not found")
		}

		newController.RunCommand(fleetName, commandFlag)

		utils.Log.Info("Command executed on fleet " + fleetName)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP("name", "n", "pwn", "Fleet name")
	runCmd.Flags().StringP("command", "c", "", "Command to send")
	runCmd.Flags().IntP("port", "p", -1, "SSH port")
	runCmd.Flags().StringP("username", "U", "", "SSH username")
	runCmd.Flags().StringP("provider", "P", "", "Service provider")

	runCmd.MarkFlagRequired("command")
	runCmd.MarkFlagRequired("name")
}
