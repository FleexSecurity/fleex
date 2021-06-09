package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/linode"
)

// imagesCmd represents the images command
var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "List available images",
	Run: func(cmd *cobra.Command, args []string) {
		provider := viper.GetString("provider")
		linodeToken := viper.GetString("linode-token")

		if strings.ToLower(provider) == "linode" {
			linode.ListImages(linodeToken)
			return
		}

		if strings.ToLower(provider) == "digitalocean" {
			// todo
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(imagesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// imagesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// imagesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
