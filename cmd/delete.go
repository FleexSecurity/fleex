package cmd

import (
	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an existing fleet or even a single box",
	Run: func(cmd *cobra.Command, args []string) {
		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		name, _ := cmd.Flags().GetString("name")
		providerFlag, _ := cmd.Flags().GetString("provider")

		if providerFlag != "" {
			globalConfig.Settings.Provider = providerFlag
		}

		provider := controller.GetProvider(globalConfig.Settings.Provider)
		if provider == -1 {
			utils.Log.Fatal(models.ErrInvalidProvider)
		}

		newController := controller.NewController(globalConfig)

		newController.DeleteFleet(name)

	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringP("name", "n", "pwn", "Fleet name. Boxes will be named [name]-[number]")
	deleteCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")

}
