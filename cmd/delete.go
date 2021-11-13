package cmd

import (
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/FleexSecurity/fleex/provider/controller"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an existing fleet or even a single box",
	Run: func(cmd *cobra.Command, args []string) {
		var token string

		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		name, _ := cmd.Flags().GetString("name")
		providerFlag, _ := cmd.Flags().GetString("provider")
		if providerFlag != "" {
			viper.Set("provider", providerFlag)
		}
		provider := controller.GetProvider(viper.GetString("provider"))

		switch provider {
		case controller.PROVIDER_LINODE:
			token = viper.GetString("linode.token")
		case controller.PROVIDER_DIGITALOCEAN:
			token = viper.GetString("digitalocean.token")
		case controller.PROVIDER_VULTR:
			token = viper.GetString("vultr.token")
		}

		controller.DeleteFleet(name, token, provider)

	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringP("name", "n", "pwn", "Fleet name. Boxes will be named [name]-[number]")
	deleteCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")

}
