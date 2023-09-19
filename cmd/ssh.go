package cmd

import (
	"log"

	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
)

// sshCmd represents the ssh command
var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Start SSH terminal for a box",

	Run: func(cmd *cobra.Command, args []string) {
		var token string

		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		portFlag, _ := cmd.Flags().GetInt("port")
		usernameFlag, _ := cmd.Flags().GetString("username")

		if globalConfig.Settings.Provider != providerFlag && providerFlag == "" {
			providerFlag = globalConfig.Settings.Provider
		}

		provider := controller.GetProvider(providerFlag)
		if provider == -1 {
			log.Fatal("invalid provider")
		}
		if portFlag == -1 {
			portFlag = globalConfig.Providers[providerFlag].Port
		}
		if usernameFlag == "" {
			usernameFlag = globalConfig.Providers[providerFlag].Username
		}
		token = globalConfig.Providers[providerFlag].Token
		boxName, _ := cmd.Flags().GetString("name")
		sshKey := globalConfig.SSHKeys.PrivateFile

		newController := controller.NewController(globalConfig)
		newController.SSH(boxName, usernameFlag, portFlag, sshKey, token, provider)
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
	sshCmd.Flags().StringP("name", "n", "pwn", "Box name")
	sshCmd.Flags().StringP("username", "U", "", "SSH username")
	sshCmd.Flags().IntP("port", "", -1, "SSH port")
	sshCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")

}
