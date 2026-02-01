package cmd

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
)

// imagesCmd represents the images command
var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Show image options",
	// Run: func(cmd *cobra.Command, args []string) {},
}

var imagesListCmd = &cobra.Command{
	Use:   "ls",
	Short: "List available images",
	Run: func(cmd *cobra.Command, args []string) {
		var token string

		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")

		if providerFlag != "" {
			globalConfig.Settings.Provider = providerFlag
		}
		providerFlag = globalConfig.Settings.Provider

		provider := controller.GetProvider(providerFlag)
		if provider == -1 {
			utils.Log.Fatal(models.ErrInvalidProvider)
		}

		newController := controller.NewController(globalConfig)
		newController.ListImages(token, provider)
	},
}

var imagesRemoveCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove images",
	Run: func(cmd *cobra.Command, args []string) {
		var token string

		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		nameFlag, _ := cmd.Flags().GetString("name")

		if providerFlag != "" {
			globalConfig.Settings.Provider = providerFlag
		}

		provider := controller.GetProvider(globalConfig.Settings.Provider)
		if provider == -1 {
			log.Fatal("provider non valido")
		}

		newController := controller.NewController(globalConfig)
		newController.RemoveImages(token, provider, nameFlag)
	},
}

var imagesTransferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Transfer an image to another region (DigitalOcean only)",
	Run: func(cmd *cobra.Command, args []string) {
		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		imageFlag, _ := cmd.Flags().GetString("image")
		regionFlag, _ := cmd.Flags().GetString("region")

		if imageFlag == "" {
			utils.Log.Fatal("image ID is required (-I or --image)")
		}
		if regionFlag == "" {
			utils.Log.Fatal("target region is required (-R or --region)")
		}

		if providerFlag != "" {
			globalConfig.Settings.Provider = providerFlag
		}

		provider := controller.GetProvider(globalConfig.Settings.Provider)
		if provider == -1 {
			utils.Log.Fatal(models.ErrInvalidProvider)
		}

		imageID, err := strconv.Atoi(imageFlag)
		if err != nil {
			utils.Log.Fatal("image ID must be a number")
		}

		newController := controller.NewController(globalConfig)
		newController.TransferImage(imageID, regionFlag)
	},
}

var imagesRegionsCmd = &cobra.Command{
	Use:   "regions",
	Short: "Show regions where an image is available (DigitalOcean only)",
	Run: func(cmd *cobra.Command, args []string) {
		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		imageFlag, _ := cmd.Flags().GetString("image")

		if imageFlag == "" {
			utils.Log.Fatal("image ID is required (-I or --image)")
		}

		if providerFlag != "" {
			globalConfig.Settings.Provider = providerFlag
		}

		provider := controller.GetProvider(globalConfig.Settings.Provider)
		if provider == -1 {
			utils.Log.Fatal(models.ErrInvalidProvider)
		}

		imageID, err := strconv.Atoi(imageFlag)
		if err != nil {
			utils.Log.Fatal("image ID must be a number")
		}

		newController := controller.NewController(globalConfig)
		regions := newController.GetImageRegions(imageID)
		fmt.Println(strings.Join(regions, ","))
	},
}

func init() {
	rootCmd.AddCommand(imagesCmd)

	imagesCmd.AddCommand(imagesListCmd)
	imagesListCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")

	imagesCmd.AddCommand(imagesRemoveCmd)
	imagesRemoveCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")
	imagesRemoveCmd.Flags().StringP("name", "n", "pwn", "Fleet name.")

	imagesCmd.AddCommand(imagesTransferCmd)
	imagesTransferCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: digitalocean)")
	imagesTransferCmd.Flags().StringP("image", "I", "", "Image ID to transfer")
	imagesTransferCmd.Flags().StringP("region", "R", "", "Target region slug (e.g., nyc1, sfo1, fra1)")

	imagesCmd.AddCommand(imagesRegionsCmd)
	imagesRegionsCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: digitalocean)")
	imagesRegionsCmd.Flags().StringP("image", "I", "", "Image ID to query")
}
