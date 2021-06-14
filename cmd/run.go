package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/controller"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a command",
	Run: func(cmd *cobra.Command, args []string) {
		var token, username, password string
		var port int
		fleetName, _ := cmd.Flags().GetString("name")
		command, _ := cmd.Flags().GetString("command")

		provider := controller.GetProvider(viper.GetString("provider"))

		switch provider {
		case controller.PROVIDER_LINODE:
			token = viper.GetString("linode.token")
			port = viper.GetInt("linode.port")
			username = viper.GetString("linode.username")
			password = viper.GetString("linode.password")
		case controller.PROVIDER_DIGITALOCEAN:
			token = viper.GetString("digitalocean.token")
			port = viper.GetInt("digitalocean.port")
			username = viper.GetString("digitalocean.username")
			password = viper.GetString("digitalocean.password")
		}

		controller.RunCommand(fleetName, command, token, port, username, password, provider)

	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringP("name", "n", "pwn", "Box name")
	runCmd.Flags().StringP("command", "c", "whoami", "Command to send")
	/* TODO: cli override for yaml settings
	scanCmd.Flags().IntP("port", "P", 2266, "SSH port")
	scanCmd.Flags().StringP("username", "u", "op", "SSH username")
	scanCmd.Flags().StringP("password", "p", "1337superPass", "SSH password")*/
}
