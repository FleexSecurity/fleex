package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/controller"
	"github.com/sw33tLie/fleex/pkg/linode"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Distributed scanning",
	Run: func(cmd *cobra.Command, args []string) {
		var token string

		command, _ := cmd.Flags().GetString("command")
		delete, _ := cmd.Flags().GetBool("delete")
		fleetName, _ := cmd.Flags().GetString("name")
		input, _ := cmd.Flags().GetString("input")
		output, _ := cmd.Flags().GetString("output")

		provider := controller.GetProvider(viper.GetString("provider"))

		switch provider {
		case controller.PROVIDER_LINODE:
			token = viper.GetString("linode.token")
			linode.Scan(fleetName, command, delete, input, output, token)

		case controller.PROVIDER_DIGITALOCEAN:
			token = viper.GetString("digitalocean.token")
			// TODO
		}

		controller.RunCommand(fleetName, command, token, provider)
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// scanCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// scanCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	scanCmd.Flags().StringP("name", "n", "pwn", "Fleet name")
	scanCmd.Flags().StringP("command", "c", "whoami", "Command to send. Supports {{INPUT}} and {{OUTPUT}}")
	scanCmd.Flags().StringP("input", "i", "", "Input file")
	scanCmd.Flags().StringP("output", "o", "scan-results.txt", "Output file path")
	scanCmd.Flags().BoolP("delete", "d", false, "Delete boxes as soon as they finish their job")

}
