package cmd

import (
	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// imagesCmd represents the images command
var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Show image options",
	// Run: func(cmd *cobra.Command, args []string) {},
}

var imagesListCmd = &cobra.Command{
	Use:   "ls",
	Short: "List available images",
	Run: func(cmd *cobra.Command, args []string) {
		var token string

		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

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

		controller.ListImages(token, provider)
	},
}

var imagesRemoveCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove images",
	Run: func(cmd *cobra.Command, args []string) {
		var token string

		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		nameFlag, _ := cmd.Flags().GetString("name")
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

		controller.RemoveImages(token, provider, nameFlag)
	},
}

func init() {
	rootCmd.AddCommand(imagesCmd)

	imagesCmd.AddCommand(imagesListCmd)
	imagesListCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")

	imagesCmd.AddCommand(imagesRemoveCmd)
	imagesRemoveCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")
	imagesRemoveCmd.Flags().StringP("name", "n", "pwn", "Fleet name.")

}
