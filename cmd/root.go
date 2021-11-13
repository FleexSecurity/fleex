package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/FleexSecurity/fleex/provider/services"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

type ProviderController struct {
	Service services.LinodeService
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "fleex",
	Short: `
███████╗██╗     ███████╗███████╗██╗  ██╗
██╔════╝██║     ██╔════╝██╔════╝╚██╗██╔╝
█████╗  ██║     █████╗  █████╗   ╚███╔╝ 
██╔══╝  ██║     ██╔══╝  ██╔══╝   ██╔██╗ 
██║     ███████╗███████╗███████╗██╔╝ ██╗
╚═╝     ╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝

Distributed computing using Linode/Digitalocean boxes.
Check out our docs at https://fleexsecurity.github.io/fleex-docs/
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	/*Run: func(cmd *cobra.Command, args []string) {

	},*/
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/fleex/config.yaml)")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().StringP("loglevel", "l", "info", "Set log level. Available: debug, info, warn, error, fatal")
	rootCmd.PersistentFlags().StringP("proxy", "", "", "HTTP Proxy (Useful for debugging. Example: http://127.0.0.1:8080)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	if cfgFile != "" {
		// Use config file from the flag.
		if !utils.FileExists(cfgFile) {
			utils.Log.Fatal("Invalid config file path")
		}

		viper.SetConfigFile(cfgFile)

	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(filepath.Join(home, "fleex"))
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		//fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
	levelString, _ := rootCmd.PersistentFlags().GetString("loglevel")
	utils.SetLogLevel(levelString)
}
