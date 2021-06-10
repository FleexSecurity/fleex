package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// initCmd represents the init command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "fleex config setup",
	Long:  "fleex config setup",
}

var configInit = &cobra.Command{
	Use:   "init",
	Short: "fleex init project",
	Long:  "fleex init project",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := rootCmd.PersistentFlags().GetString("config")
		fmt.Println(configPath)

		viper.SetConfigType("yaml")
		viper.SetDefault("provider", []string{"linode", "digitalocean"})
		viper.SetDefault("linode-image", "{YOUR LINODE IMAGE}")
		viper.SetDefault("linode-region", "{YOUR REGION}")
		viper.SetDefault("linode-token", "{YOUR TOKEN}")
		err := viper.SafeWriteConfigAs(configPath)
		if err != nil {
			fmt.Println(err)
		}
	},
}

var configGet = &cobra.Command{
	Use:   "get",
	Short: "fleex get data from config file",
	Long:  "fleex get data from config file",
	Run: func(cmd *cobra.Command, args []string) {
		fieldFlag, _ := cmd.Flags().GetString("field")
		viper.SetConfigType("yaml")
		viper.ReadInConfig()

		if strings.Contains(fieldFlag, ",") {
			fields := strings.Split(fieldFlag, ",")
			for _, singleField := range fields {
				field := viper.Get(singleField)
				fmt.Println("-", singleField, ":", field)
			}
		} else {
			fmt.Println("-", fieldFlag, ":", viper.Get(fieldFlag))
		}
	},
}

var configSet = &cobra.Command{
	Use:   "set",
	Short: "fleex set data in config file",
	Long:  "fleex set data in config file",
	Run: func(cmd *cobra.Command, args []string) {
		key, _ := cmd.Flags().GetString("key")
		value, _ := cmd.Flags().GetString("value")
		viper.SetConfigType("yaml")
		viper.Set(key, value)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInit)
	configCmd.AddCommand(configGet)
	configCmd.AddCommand(configSet)

	configGet.Flags().StringP("field", "f", "", "field to retrieve, comma separated")
	configSet.Flags().StringP("key", "k", "", "key")
	configSet.Flags().StringP("value", "v", "", "value")

}
