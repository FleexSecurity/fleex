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
	Short: "Config setup",
	Long:  "Config setup",
}

var configGet = &cobra.Command{
	Use:   "get",
	Short: "Get data from config file",
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

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGet)

	configGet.Flags().StringP("field", "f", "provider", "field to retrieve, comma separated")
}
