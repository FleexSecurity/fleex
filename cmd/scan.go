package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/controller"
	scan "github.com/sw33tLie/fleex/pkg/scan"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Distributed scanning",
	Run: func(cmd *cobra.Command, args []string) {
		var token string

		commandFlag, _ := cmd.Flags().GetString("command")
		deleteFlag, _ := cmd.Flags().GetBool("delete")
		concat, _ := cmd.Flags().GetBool("concat")
		fleetNameFlag, _ := cmd.Flags().GetString("name")
		inputFlag, _ := cmd.Flags().GetString("input")
		outputFlag, _ := cmd.Flags().GetString("output")

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

		scan.Start(fleetNameFlag, commandFlag, deleteFlag, inputFlag, outputFlag, concat, token, port, username, password, provider)

	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().StringP("name", "n", "pwn", "Fleet name")
	scanCmd.Flags().StringP("command", "c", "whoami", "Command to send. Supports {{INPUT}} and {{OUTPUT}}")
	scanCmd.Flags().StringP("input", "i", "", "Input file")
	scanCmd.Flags().StringP("output", "o", "", "Output file path. If empty, will use ~/fleex/<unix_timestamp>/")
	scanCmd.Flags().StringP("provider", "p", "", "VPS provider (Supported: linode, digitalocean)")
	scanCmd.Flags().IntP("port", "", 2266, "SSH port")
	scanCmd.Flags().StringP("username", "U", "op", "SSH username")
	scanCmd.Flags().StringP("password", "P", "1337superPass", "SSH password")
	scanCmd.Flags().BoolP("delete", "d", false, "Delete boxes as soon as they finish their job")
	scanCmd.Flags().BoolP("concat", "", false, "Store a single file only, made from concatenating all output chunks")
}
