package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/controller"
)

// spawnCmd represents the spawn command
var spawnCmd = &cobra.Command{
	Use:   "spawn",
	Short: "Spawn a fleet",
	Run: func(cmd *cobra.Command, args []string) {
		var token, image, region, size string
		var digSlug string

		provider := controller.GetProvider(viper.GetString("provider"))

		fleetCount, _ := cmd.Flags().GetInt("count")
		fleetName, _ := cmd.Flags().GetString("name")
		waitFlag, _ := cmd.Flags().GetBool("wait")

		switch provider {
		case controller.PROVIDER_LINODE:
			token = viper.GetString("linode.token")
			region = viper.GetString("linode.region")
			image = viper.GetString("linode.image")
			size = viper.GetString("linode.size")

		case controller.PROVIDER_DIGITALOCEAN:
			token = viper.GetString("digitalocean.token")
			region = viper.GetString("digitalocean.region")
			image = viper.GetString("digitalocean.image-id")
			size = viper.GetString("digitalocean.size")
			digSlug = viper.GetString("digitalocean.slug")
		}

		fmt.Println(size, digSlug)
		controller.SpawnFleet(fleetName, fleetCount, image, region, token, waitFlag, provider)

	},
}

func init() {
	rootCmd.AddCommand(spawnCmd)
	spawnCmd.Flags().IntP("count", "c", 2, "How many box to spawn")
	spawnCmd.Flags().StringP("name", "n", "pwn", "Fleet name. Boxes will be named [name]-[number]")
	spawnCmd.Flags().BoolP("wait", "w", false, "Wait until all boxes are running")
}
