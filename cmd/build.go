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
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"github.com/hnakamur/go-scp"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/controller"
	"github.com/sw33tLie/fleex/pkg/digitalocean"
	"github.com/sw33tLie/fleex/pkg/sshutils"
	"gopkg.in/yaml.v2"
)

type BuildConfig struct {
	//Name   string
	Config struct {
		Source      string `yaml:"source"`
		Destination string `yaml:"destination"`
	}

	Commands []string
}

type myData struct {
	Conf struct {
		Hits      int64
		Time      int64
		CamelCase string `yaml:"camelCase"`
	}
}

var data = `
config:
  source: ./configs
  destination: /tmp/configs
commands:
  - fallocate -l 2G /swap && chmod 600 /swap && mkswap /swap && swapon /swap
`

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build image",
	Long:  "Build image",
	Run: func(cmd *cobra.Command, args []string) {
		timeNow := strconv.FormatInt(time.Now().Unix(), 10)
		fleetName := "fleex-" + timeNow
		boxID := 0
		boxIP := ""
		// image := viper.GetString("digitalocean.image")
		region := viper.GetString("digitalocean.region")
		size := viper.GetString("digitalocean.size")
		publicSSH := viper.GetString("public-ssh-file")
		sshFingerprint := sshutils.SSHFingerprintGen(publicSSH)
		tags := []string{"snapshot"}
		token := viper.GetString("digitalocean.token")

		providerFlag, _ := cmd.Flags().GetString("provider")
		fileFlag, _ := cmd.Flags().GetString("file")

		if providerFlag != "" {
			viper.Set("provider", providerFlag)
		}
		provider := controller.GetProvider(viper.GetString("provider"))

		// 1 - Spawn
		// 2 - SendDir
		// 3 - RunCommands
		// 4 - Build img

		c, err := readConf(fileFlag)
		if err != nil {
			log.Fatal(err)
		}
		// 1
		controller.SpawnFleet(fleetName, 1, "ubuntu-20-04-x64", region, size, sshFingerprint, tags, token, true, provider)

		fleets := digitalocean.GetBoxes(token)
		for _, box := range fleets {
			if box.Label == fleetName {
				boxID = box.ID
				boxIP = box.IP
			}
		}
		fmt.Println("BOXID", boxID, boxIP, fleetName)

		time.Sleep(8 * time.Second)

		err = scp.NewSCP(sshutils.GetConnection(boxIP, 22, "root", "1337superPass").Client).SendDir(c.Config.Source, c.Config.Destination, nil)
		if err != nil {
			log.Fatal(err)
		}

		for _, command := range c.Commands {
			// fmt.Println("COMMAND", command)
			controller.RunCommand(fleetName, command, token, 22, "root", "1337superPass", provider)
		}

		fmt.Println("WAIT A SECOND!")
		time.Sleep(8 * time.Second)

		digitalocean.CreateImage(token, boxID, "Fleex-build-"+timeNow)
		time.Sleep(5 * time.Second)
		fmt.Println("Delete")
		controller.DeleteFleet(fleetName, token, provider)
	},
}

func init() {
	home, _ := homedir.Dir()
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean)")
	buildCmd.Flags().StringP("file", "f", home+"/fleex/build/test.yaml", "Build file")

}

func readConf(filename string) (*BuildConfig, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &BuildConfig{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %v", filename, err)
	}

	return c, nil
}
