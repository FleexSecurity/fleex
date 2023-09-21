package cmd

import (
	"log"

	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
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
			globalConfig.Settings.Provider = providerFlag
		}

		provider := controller.GetProvider(globalConfig.Settings.Provider)
		if provider == -1 {
			utils.Log.Fatal(models.ErrInvalidProvider)
		}

		newController := controller.NewController(globalConfig)
		newController.ListImages(token, provider)
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
			globalConfig.Settings.Provider = providerFlag
		}

		provider := controller.GetProvider(globalConfig.Settings.Provider)
		if provider == -1 {
			log.Fatal("provider non valido")
		}

		newController := controller.NewController(globalConfig)
		newController.RemoveImages(token, provider, nameFlag)
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
