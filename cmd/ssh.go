package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/controller"
)

// sshCmd represents the ssh command
var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Start SSH",

	Run: func(cmd *cobra.Command, args []string) {
		var token string

		provider := controller.GetProvider(viper.GetString("provider"))

		boxName, _ := cmd.Flags().GetString("name")
		port, _ := cmd.Flags().GetInt("port")

		switch provider {
		case controller.PROVIDER_LINODE:
			token = viper.GetString("linode.token")
		case controller.PROVIDER_DIGITALOCEAN:
			token = viper.GetString("digitalocean.token")
		}

		controller.SSH(boxName, port, token, provider)
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
	sshCmd.Flags().StringP("name", "n", "pwn", "Box name")
	sshCmd.Flags().IntP("port", "", 2266, "SSH port")

}
