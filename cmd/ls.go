package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/controller"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List running boxes",
	Run: func(cmd *cobra.Command, args []string) {
		var token string

		provider := controller.GetProvider(viper.GetString("provider"))

		switch provider {
		case controller.PROVIDER_LINODE:
			token = viper.GetString("linode.token")
		case controller.PROVIDER_DIGITALOCEAN:
			token = viper.GetString("digitalocean.token")
		}

		controller.ListBoxes(token, provider)
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
