package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/services"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

var cfgFile string
var globalConfig *models.Config

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

Distributed computing using Linode/Digitalocean/Vultr boxes.
Check out our docs at https://fleexsecurity.github.io/fleex-docs/
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) {

	// },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		utils.Log.Fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
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

	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		cfgFile = filepath.Join(home, "fleex", "config.json")
	}

	file, err := os.Open(cfgFile)
	if err != nil {
		utils.Log.Fatal(err)
	}
	defer file.Close()

	var config models.Config
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		utils.Log.Fatal(err)
	}

	globalConfig = &config

	levelString, _ := rootCmd.PersistentFlags().GetString("loglevel")
	utils.SetLogLevel(levelString)
}
