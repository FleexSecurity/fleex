package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/digitalocean"
	"github.com/sw33tLie/fleex/pkg/linode"
)

// spawnCmd represents the spawn command
var spawnCmd = &cobra.Command{
	Use:   "spawn",
	Short: "Spawn a fleet",
	Run: func(cmd *cobra.Command, args []string) {
		fleetCount, _ := cmd.Flags().GetInt("count")
		fleetName, _ := cmd.Flags().GetString("name")
		//provider, _ := cmd.Flags().GetString("provider")

		provider := viper.GetString("provider")
		linodeImage := viper.GetString("linode-image")
		linodeRegion := viper.GetString("linode-region")
		linodeToken := viper.GetString("linode-token")

		fmt.Println("IMAGE: ", linodeImage)
		if strings.ToLower(provider) == "linode" {
			linode.SpawnFleet(fleetName, fleetCount, linodeImage, linodeRegion, linodeToken)
			return
		}

		if strings.ToLower(provider) == "digitalocean" {
			digitalocean.SpawnFleet(fleetName, fleetCount)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(spawnCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// spawnCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// spawnCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	spawnCmd.Flags().IntP("count", "c", 2, "How many box to spawn")
	spawnCmd.Flags().StringP("name", "n", "pwn", "Fleet name. Boxes will be named [name]-[number]")
	//spawnCmd.Flags().StringP("provider", "p", "linode", "Service provider (Supported: linode, digitalocean)")

}
