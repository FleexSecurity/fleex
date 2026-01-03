package cmd

import (
	"fmt"

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
		buildRecipe, _ := cmd.Flags().GetString("build")
		noVerify, _ := cmd.Flags().GetBool("no-verify")

		if providerFlag != "" {
			globalConfig.Settings.Provider = providerFlag
		}
		providerFlag = globalConfig.Settings.Provider

		provider := controller.GetProvider(providerFlag)
		if provider == -1 {
			utils.Log.Fatal(models.ErrInvalidProvider)
		}

		providerInfo := globalConfig.Providers[providerFlag]
		if regionFlag != "" {
			providerInfo.Region = regionFlag
		}
		if sizeFlag != "" {
			providerInfo.Size = sizeFlag
		}
		if imageFlag != "" {
			providerInfo.Image = imageFlag
		}

		newController := controller.NewController(globalConfig)
		newController.SpawnFleet(fleetName, fleetCount, skipWait, false)

		if buildRecipe != "" {
			recipe, err := utils.ReadBuildFile(buildRecipe)
			if err != nil {
				utils.Log.Fatal("Failed to load build recipe: ", err)
			}

			fmt.Printf("Building fleet '%s' with recipe '%s'...\n", fleetName, recipe.Name)

			opts := models.BuildOptions{
				Recipe:    recipe,
				FleetName: fleetName,
				Parallel:  5,
				NoVerify:  noVerify,
				Verbose:   true,
			}

			results, err := newController.BuildFleet(opts)
			if err != nil {
				utils.Log.Fatal(err)
			}

			successCount := 0
			for _, r := range results {
				if r.Success {
					successCount++
				}
			}

			fmt.Printf("Build complete: %d/%d successful\n", successCount, len(results))
		}
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
	spawnCmd.Flags().StringP("build", "b", "", "Build recipe to run after spawn")
	spawnCmd.Flags().BoolP("no-verify", "", false, "Skip build verification")
}
