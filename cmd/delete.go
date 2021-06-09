package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/linode"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a fleet or a single box",
	Run: func(cmd *cobra.Command, args []string) {
		boxOrFleetName, _ := cmd.Flags().GetString("name")

		provider := viper.GetString("provider")
		linodeToken := viper.GetString("linode-token")

		if strings.ToLower(provider) == "linode" {
			linode.DeleteFleetOrBox(boxOrFleetName, linodeToken)
			return
		}

		if strings.ToLower(provider) == "digitalocean" {
			// todo
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deleteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// deleteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	deleteCmd.Flags().StringP("name", "n", "pwn", "Fleet name. Boxes will be named [name]-[number]")

}
