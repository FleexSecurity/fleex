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
		var token string
		fleetName, _ := cmd.Flags().GetString("name")
		command, _ := cmd.Flags().GetString("command")

		providerFlag, _ := cmd.Flags().GetString("provider")
		if providerFlag != "" {
			viper.Set("provider", providerFlag)
		}
		provider := controller.GetProvider(viper.GetString("provider"))
		providerFlag = viper.GetString("provider")

		portFlag, _ := cmd.Flags().GetInt("port")
		usernameFlag, _ := cmd.Flags().GetString("username")
		passwordFlag, _ := cmd.Flags().GetString("password")
		if portFlag != 0 {
			viper.Set(providerFlag+".port", portFlag)
		}
		if usernameFlag != "" {
			viper.Set(providerFlag+".username", usernameFlag)
		}
		if passwordFlag != "" {
			viper.Set(providerFlag+".password", passwordFlag)
		}

		port := viper.GetInt(providerFlag + ".port")
		username := viper.GetString(providerFlag + ".username")
		password := viper.GetString(providerFlag + ".password")

		switch provider {
		case controller.PROVIDER_LINODE:
			token = viper.GetString("linode.token")
		case controller.PROVIDER_DIGITALOCEAN:
			token = viper.GetString("digitalocean.token")
		}

		// log.Fatalln(fleetName, command, token, port, username, password, provider)
		controller.RunCommand(fleetName, command, token, port, username, password, provider)

	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringP("name", "n", "pwn", "Box name")
	runCmd.Flags().StringP("command", "c", "whoami", "Command to send")
	// TODO: cli override for yaml settings
	runCmd.Flags().IntP("port", "", 2266, "SSH port")
	runCmd.Flags().StringP("username", "U", "op", "SSH username")
	runCmd.Flags().StringP("password", "P", "1337superPass", "SSH password")
	runCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean)")

}
