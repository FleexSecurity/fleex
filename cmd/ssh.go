package cmd

import (
	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
)

// sshCmd represents the ssh command
var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Start SSH terminal for a box",

	Run: func(cmd *cobra.Command, args []string) {
		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		boxName, _ := cmd.Flags().GetString("name")
		providerFlag, _ := cmd.Flags().GetString("provider")
		portFlag, _ := cmd.Flags().GetInt("port")
		usernameFlag, _ := cmd.Flags().GetString("username")

		if providerFlag != "" {
			globalConfig.Settings.Provider = providerFlag
		}
		providerFlag = globalConfig.Settings.Provider

		vmInfo := models.GetVMInfo(providerFlag, boxName, globalConfig)
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
		newController.SSH(boxName, vmInfo.Username, vmInfo.Password, vmInfo.Port, vmInfo.KeyPath)
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
	sshCmd.Flags().StringP("name", "n", "pwn", "Box name")
	sshCmd.Flags().StringP("username", "U", "", "SSH username")
	sshCmd.Flags().IntP("port", "", -1, "SSH port")
	sshCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")

}
