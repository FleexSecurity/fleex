package cmd

import (
	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
)

// spawnCmd represents the spawn command
var spawnCmd = &cobra.Command{
	Use:   "spawn",
	Short: "Spawn a fleet or even a single box",
	Run: func(cmd *cobra.Command, args []string) {
		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		regionFlag, _ := cmd.Flags().GetString("region")
		sizeFlag, _ := cmd.Flags().GetString("size")
		imageFlag, _ := cmd.Flags().GetString("image")
		fleetCount, _ := cmd.Flags().GetInt("count")
		fleetName, _ := cmd.Flags().GetString("name")
		skipWait, _ := cmd.Flags().GetBool("skipwait")

		if globalConfig.Settings.Provider != providerFlag && providerFlag == "" {
			providerFlag = globalConfig.Settings.Provider
		}

		provider := controller.GetProvider(providerFlag)
		if provider == -1 {
			utils.Log.Fatal(models.ErrInvalidProvider)
		}

		if regionFlag == "" {
			regionFlag = globalConfig.Providers[providerFlag].Region
		}
		if sizeFlag == "" {
			sizeFlag = globalConfig.Providers[providerFlag].Size

		}
		if imageFlag == "" {
			imageFlag = globalConfig.Providers[providerFlag].Image
		}

		newController := controller.NewController(globalConfig)
		newController.SpawnFleet(fleetName, fleetCount, skipWait, false)

	},
}

func init() {
	rootCmd.AddCommand(spawnCmd)
	spawnCmd.Flags().IntP("count", "c", 2, "How many box to spawn")
	spawnCmd.Flags().StringP("name", "n", "pwn", "Fleet name. Boxes will be named [name]-[number]")
	spawnCmd.Flags().BoolP("skipwait", "", false, "Skip waiting until all boxes are running")

	// spawnCmd.Flags().StringP("username", "U", "op", "Username")
	// spawnCmd.Flags().StringP("password", "P", "1337superPass", "Password")
	// spawnCmd.Flags().IntP("port", "", 2266, "SSH port")
	spawnCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")
	spawnCmd.Flags().StringP("region", "R", "", "Region")
	spawnCmd.Flags().StringP("size", "S", "", "Size")
	spawnCmd.Flags().StringP("image", "I", "", "Image")

	//spawnCmd.Flags().StringP("provider", "p", "linode", "Service provider (Supported: linode, digitalocean, vultr)")
}
