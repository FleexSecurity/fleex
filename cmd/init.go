/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"os"
	"strconv"
	"time"

	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "This command initializes fleex",
	Long:  `This command initializes fleex, downloads all the configuration files and puts them in $HOME/fleex`,
	Run: func(cmd *cobra.Command, args []string) {
		var fileUrl string
		linkFlag, _ := cmd.Flags().GetString("url")
		home, _ := homedir.Dir()
		timeNow := strconv.FormatInt(time.Now().Unix(), 10)
		overwrite, _ := cmd.Flags().GetBool("overwrite")

		if _, err := os.Stat(home + "/fleex"); !os.IsNotExist(err) {
			if !overwrite {
				utils.Log.Fatal("Fleex folder already exists, if you want to overwrite it use the --overwrite flag ")
			}
		}

		if linkFlag == "" {
			fileUrl = "https://github.com/FleexSecurity/fleex/releases/download/v1.0/config.zip"
		} else {
			fileUrl = linkFlag
		}
		err := utils.DownloadFile("/tmp/fleex-config-"+timeNow+".zip", fileUrl)
		if err != nil {
			panic(err)
		}
		utils.Unzip("/tmp/fleex-config-"+timeNow+".zip", home+"/fleex")
		err = os.Remove("/tmp/fleex-config-" + timeNow + ".zip")
		if err != nil {
			utils.Log.Fatal(err)
		}

		utils.Log.Info("Fleex initialized successfully, see $HOME/fleex")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringP("url", "u", "", "Config folder url")
	initCmd.Flags().BoolP("overwrite", "o", false, "If the fleex folder exists overwrite it")
}
