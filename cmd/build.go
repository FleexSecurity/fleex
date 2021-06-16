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
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/controller"
	"github.com/sw33tLie/fleex/pkg/digitalocean"
	"github.com/sw33tLie/fleex/pkg/sshutils"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build image",
	Long:  "Build image",
	Run: func(cmd *cobra.Command, args []string) {
		fleetName := "fleex"
		image := viper.GetString("digitalocean.image")
		region := viper.GetString("digitalocean.region")
		size := viper.GetString("digitalocean.size")
		publicSSH := viper.GetString("public-ssh-file")
		sshFingerprint := sshutils.SSHFingerprintGen(publicSSH)
		tags := []string{"snapshot"}
		token := viper.GetString("digitalocean.token")

		providerFlag, _ := cmd.Flags().GetString("provider")
		if providerFlag != "" {
			viper.Set("provider", providerFlag)
		}
		provider := controller.GetProvider(viper.GetString("provider"))
		controller.SpawnFleet(fleetName, 1, image, region, size, sshFingerprint, tags, token, true, provider)
		file, err := os.Open("./testbuild.txt")
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
			controller.RunCommand(fleetName+"-1", scanner.Text(), token, 22, "root", "1337superPass", provider)
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		//linode.CreateImage(token, 1, "", "")
		digitalocean.CreateImage(token, 1, "Fleex-build")

	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean)")

}
