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
	"path/filepath"
	"strconv"
	"time"

	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Fleex initialization command. Run this the first time.",
	Run: func(cmd *cobra.Command, args []string) {
		linkFlag, _ := cmd.Flags().GetString("url")
		overwrite, _ := cmd.Flags().GetBool("overwrite")

		configDir, err := utils.GetConfigDir()
		if err != nil {
			utils.Log.Fatal(err)
		}

		if _, err := os.Stat(configDir + "/fleex"); !os.IsNotExist(err) {
			if !overwrite {
				utils.Log.Fatal("Fleex folder already exists, if you want to overwrite it use the --overwrite flag ")
			}
		}

		fileUrl := "https://github.com/FleexSecurity/fleex/releases/download/v1.0/config.zip"
		if linkFlag != "" {
			fileUrl = linkFlag
		}

		timeNow := strconv.FormatInt(time.Now().Unix(), 10)
		tmpZipPath := filepath.Join("/tmp", "fleex-config-"+timeNow+".zip")

		err = utils.DownloadFile(tmpZipPath, fileUrl)
		if err != nil {
			utils.Log.Fatal(err)
		}

		destPath := filepath.Join(configDir, "fleex")
		err = utils.Unzip(tmpZipPath, destPath)
		if err != nil {
			utils.Log.Fatal(err)
		}

		err = os.Remove(tmpZipPath)
		if err != nil {
			utils.Log.Fatal(err)
		}

		utils.Log.Info("Fleex initialized successfully, see", destPath)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringP("url", "u", "", "Config folder url")
	initCmd.Flags().BoolP("overwrite", "o", false, "If the fleex folder exists overwrite it")
}
