package cmd

import (
	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
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
		if providerFlag != "" {
			globalConfig.Settings.Provider = providerFlag
		}

		provider := controller.GetProvider(globalConfig.Settings.Provider)
		if provider == -1 {
			utils.Log.Fatal(models.ErrInvalidProvider)
		}

		token = globalConfig.Providers[providerFlag].Token

		newController := controller.NewController(globalConfig)
		newController.ListBoxes(token, provider)
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)

	lsCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")

}
