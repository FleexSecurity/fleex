package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/controller"
	"github.com/sw33tLie/fleex/pkg/sshutils"
)

// spawnCmd represents the spawn command
var spawnCmd = &cobra.Command{
	Use:   "spawn",
	Short: "Spawn a fleet",
	Run: func(cmd *cobra.Command, args []string) {
		var token, image, region, size, sshFingerprint string
		var tags []string

		providerFlag, _ := cmd.Flags().GetString("provider")
		regionFlag, _ := cmd.Flags().GetString("region")
		sizeFlag, _ := cmd.Flags().GetString("size")
		imageFlag, _ := cmd.Flags().GetString("image")

		if providerFlag != "" {
			viper.Set("provider", providerFlag)
		}
		provider := controller.GetProvider(viper.GetString("provider"))
		providerFlag = viper.GetString("provider")
		publicSSH := viper.GetString("public-ssh-file")

		fleetCount, _ := cmd.Flags().GetInt("count")
		fleetName, _ := cmd.Flags().GetString("name")
		waitFlag, _ := cmd.Flags().GetBool("wait")

		if regionFlag != "" {
			viper.Set(providerFlag+".region", regionFlag)
		}
		if sizeFlag != "" {
			viper.Set(providerFlag+".size", sizeFlag)
		}
		if imageFlag != "" {
			viper.Set(providerFlag+".image", imageFlag)
		}

		switch provider {
		case controller.PROVIDER_LINODE:
			token = viper.GetString("linode.token")
			region = viper.GetString("linode.region")
			image = viper.GetString("linode.image")
			size = viper.GetString("linode.size")
			sshFingerprint = "" // not needed on Linode

		case controller.PROVIDER_DIGITALOCEAN:
			token = viper.GetString("digitalocean.token")
			region = viper.GetString("digitalocean.region")
			image = viper.GetString("digitalocean.image")
			size = viper.GetString("digitalocean.size")
			sshFingerprint = sshutils.SSHFingerprintGen(publicSSH)
			tags = viper.GetStringSlice("digitalocean.tags")
		}
		controller.SpawnFleet(fleetName, fleetCount, image, region, size, sshFingerprint, tags, token, waitFlag, provider)

	},
}

func init() {
	rootCmd.AddCommand(spawnCmd)
	spawnCmd.Flags().IntP("count", "c", 2, "How many box to spawn")
	spawnCmd.Flags().StringP("name", "n", "pwn", "Fleet name. Boxes will be named [name]-[number]")
	spawnCmd.Flags().BoolP("wait", "w", false, "Wait until all boxes are running")

	// spawnCmd.Flags().StringP("username", "U", "op", "Username")
	// spawnCmd.Flags().StringP("password", "P", "1337superPass", "Password")
	// spawnCmd.Flags().IntP("port", "", 2266, "SSH port")
	spawnCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean)")
	spawnCmd.Flags().StringP("region", "R", "", "Region")
	spawnCmd.Flags().StringP("size", "S", "", "Size")
	spawnCmd.Flags().StringP("image", "I", "", "Image")

	//spawnCmd.Flags().StringP("provider", "p", "linode", "Service provider (Supported: linode, digitalocean)")
}
