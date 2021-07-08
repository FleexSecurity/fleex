package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/controller"
	"github.com/sw33tLie/fleex/pkg/utils"
)

// sshCmd represents the ssh command
var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Start SSH",

	Run: func(cmd *cobra.Command, args []string) {
		var token string
		var port int

		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		portFlag, _ := cmd.Flags().GetInt("port")
		username, _ := cmd.Flags().GetString("username")

		if providerFlag != "" {
			viper.Set("provider", providerFlag)
		}
		provider := controller.GetProvider(viper.GetString("provider"))
		providerFlag = viper.GetString("provider")

		if portFlag != -1 {
			viper.Set(providerFlag+".port", portFlag)
		}
		if username != "" {
			viper.Set(providerFlag+".username", username)
		}

		boxName, _ := cmd.Flags().GetString("name")

		sshKey := viper.GetString("private-ssh-file")

		switch provider {
		case controller.PROVIDER_LINODE:
			token = viper.GetString("linode.token")
			port = viper.GetInt("linode.port")
			username = viper.GetString("linode.username")
		case controller.PROVIDER_DIGITALOCEAN:
			token = viper.GetString("digitalocean.token")
			port = viper.GetInt("digitalocean.port")
			username = viper.GetString("digitalocean.username")
		}

		fmt.Println("port", port, "username", username, providerFlag)
		log.Fatal(1)

		controller.SSH(boxName, username, port, sshKey, token, provider)
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
	sshCmd.Flags().StringP("name", "n", "pwn", "Box name")
	sshCmd.Flags().StringP("username", "u", "", "SSH username")
	sshCmd.Flags().IntP("port", "", -1, "SSH port")
	sshCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean)")

}
