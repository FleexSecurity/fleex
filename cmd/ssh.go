package cmd

import (
	"fmt"

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
		boxName, _ := cmd.Flags().GetString("name")
		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		portFlag, _ := cmd.Flags().GetInt("port")
		usernameFlag, _ := cmd.Flags().GetString("username")

		if providerFlag != "" {
			globalConfig.Settings.Provider = providerFlag
		}
		providerFlag = globalConfig.Settings.Provider
		sshKey := globalConfig.SSHKeys.PrivateFile

		if globalConfig.Settings.Provider == "custom" {
			var customVps *models.CustomVM
			found := false

			for _, vps := range globalConfig.CustomVMs {
				if vps.InstanceID == boxName {
					customVps = &vps
					found = true
					break
				}
			}

			if !found {
				utils.Log.Fatal("Error: CustomVps with the specified InstanceID not found.")
			}

			if portFlag != -1 {
				customVps.SSHPort = portFlag
			}
			if usernameFlag != "" {
				customVps.Username = usernameFlag
			}

			newController := controller.NewController(globalConfig)
			newController.SSH(boxName, customVps.Username, customVps.Password, customVps.SSHPort, sshKey)
		} else {
			provider := controller.GetProvider(providerFlag)
			fmt.Println(provider, providerFlag)
			if provider == -1 {
				utils.Log.Fatal(models.ErrInvalidProvider)
			}

			providerInfo := globalConfig.Providers[providerFlag]
			if portFlag == -1 {
				providerInfo.Port = portFlag
			}
			if usernameFlag != "" {
				providerInfo.Username = usernameFlag
			}
			password := providerInfo.Password
			globalConfig.Providers[providerFlag] = providerInfo

			sshKey := globalConfig.SSHKeys.PrivateFile

			newController := controller.NewController(globalConfig)
			newController.SSH(boxName, usernameFlag, password, portFlag, sshKey)
		}
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
	sshCmd.Flags().StringP("name", "n", "pwn", "Box name")
	sshCmd.Flags().StringP("username", "U", "", "SSH username")
	sshCmd.Flags().IntP("port", "", -1, "SSH port")
	sshCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")

}
