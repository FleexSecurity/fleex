package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/digitalocean"
	"github.com/sw33tLie/fleex/pkg/linode"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a command",
	Run: func(cmd *cobra.Command, args []string) {
		boxName, _ := cmd.Flags().GetString("name")
		command, _ := cmd.Flags().GetString("command")

		provider := viper.GetString("provider")
		linodeToken := viper.GetString("linode.token")
		digitaloceanToken := viper.GetString("digitalocean.token")
		doSshUser := viper.GetString("digitalocean.username")
		doSshPort := viper.GetInt("digitalocean.port")
		doSshPassword := viper.GetString("digitalocean.password")

		if strings.ToLower(provider) == "linode" {
			linode.RunCommand(boxName, command, linodeToken)
			return
		}

		if strings.ToLower(provider) == "digitalocean" {
			digitalocean.RunCommand(boxName, command, digitaloceanToken, doSshPort, doSshUser, doSshPassword)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringP("name", "n", "pwn", "Box name")
	runCmd.Flags().StringP("command", "c", "whoami", "Command to send")
}
