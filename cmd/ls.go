package cmd

import (
	"log"

	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List running boxes",
	Run: func(cmd *cobra.Command, args []string) {
		var token string

		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		if globalConfig.Settings.Provider != providerFlag && providerFlag == "" {
			providerFlag = globalConfig.Settings.Provider
		}

		provider := controller.GetProvider(providerFlag)
		if provider == -1 {
			log.Fatal("invalid provider")
		}

		token = globalConfig.Providers[providerFlag].Token
		controller.ListBoxes(token, provider)
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)

	lsCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")

}
