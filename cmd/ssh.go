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
		var port int
		providerFlag, _ := cmd.Flags().GetString("provider")
		portFlag, _ := cmd.Flags().GetInt("port")
		boxName, _ := cmd.Flags().GetString("name")

		if providerFlag != "" {
			viper.Set("provider", providerFlag)
		}
		provider := controller.GetProvider(viper.GetString("provider"))
		providerFlag = viper.GetString("provider")

		if portFlag != 2266 {
			viper.Set(providerFlag+".port", portFlag)
		} else {
			viper.Set(providerFlag+".port", 2266)
		}

		switch provider {
		case controller.PROVIDER_LINODE:
			token = viper.GetString("linode.token")
			port = viper.GetInt("linode.port")
		case controller.PROVIDER_DIGITALOCEAN:
			token = viper.GetString("digitalocean.token")
			port = viper.GetInt("digitalocean.port")
		}

		controller.SSH(boxName, port, token, provider)
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
	sshCmd.Flags().StringP("name", "n", "pwn", "Box name")
	sshCmd.Flags().IntP("port", "", 2266, "SSH port")
	sshCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean)")

}
