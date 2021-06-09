package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/linode"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		boxName, _ := cmd.Flags().GetString("name")
		command, _ := cmd.Flags().GetString("command")

		provider := viper.GetString("provider")
		linodeToken := viper.GetString("linode-token")

		if strings.ToLower(provider) == "linode" {
			linode.RunCommand(boxName, command, linodeToken)
			return
		}

		if strings.ToLower(provider) == "digitalocean" {
			// todo
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	runCmd.Flags().StringP("name", "n", "pwn", "Box name")
	runCmd.Flags().StringP("command", "c", "whoami", "Command to send")

}
