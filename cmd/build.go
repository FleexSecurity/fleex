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
	"strings"
	"time"

	"github.com/hnakamur/go-scp"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/controller"
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

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build image",
	Long:  "Build image",
	Run: func(cmd *cobra.Command, args []string) {
		var token, region, size, sshFingerprint, boxIP string
		var boxID int
		timeNow := strconv.FormatInt(time.Now().Unix(), 10)
		home, _ := homedir.Dir()
		fleetName := "fleex-" + timeNow
		// boxID := 0
		// boxIP := ""
		publicSSH := viper.GetString("public-ssh-file")
		tags := []string{"snapshot"}

		providerFlag, _ := cmd.Flags().GetString("provider")
		regionFlag, _ := cmd.Flags().GetString("region")
		sizeFlag, _ := cmd.Flags().GetString("size")
		fileFlag, _ := cmd.Flags().GetString("file")
		deleteFlag, _ := cmd.Flags().GetBool("delete")

		if providerFlag != "" {
			viper.Set("provider", providerFlag)
		}
		provider := controller.GetProvider(viper.GetString("provider"))

		if regionFlag != "" {
			viper.Set(providerFlag+".region", regionFlag)
		}
		if sizeFlag != "" {
			viper.Set(providerFlag+".size", regionFlag)
		}

		// log.Fatal(deleteFlag, provider, providerFlag)

		switch provider {
		case controller.PROVIDER_LINODE:
			token = viper.GetString("linode.token")
			region = viper.GetString("linode.region")
			size = viper.GetString("linode.size")
		case controller.PROVIDER_DIGITALOCEAN:
			token = viper.GetString("digitalocean.token")
			region = viper.GetString("digitalocean.region")
			size = viper.GetString("digitalocean.size")
			sshFingerprint = sshutils.SSHFingerprintGen(publicSSH)
		}

		c, err := readConf(fileFlag)
		if err != nil {
			log.Fatal(err)
		}
		controller.SpawnFleet(fleetName, 1, "ubuntu-20-04-x64", region, size, sshFingerprint, tags, token, true, provider)

		fleets := controller.GetFleet(fleetName, token, provider)
		for _, box := range fleets {
			if box.Label == fleetName {
				boxID = box.ID
				boxIP = box.IP
				break
			}
		}

		time.Sleep(20 * time.Second)

		if strings.ContainsAny("~", c.Config.Source) {
			c.Config.Source = strings.ReplaceAll(c.Config.Source, "~", home)
		}
		//fmt.Println("SOURCE:", c.Config.Source)
		err = scp.NewSCP(sshutils.GetConnection(boxIP, 22, "root", "1337superPass").Client).SendDir(c.Config.Source, c.Config.Destination, nil)
		if err != nil {
			log.Fatal(err)
		}

		// log.Fatal(1)

		for _, command := range c.Commands {
			controller.RunCommand(fleetName, command, token, 22, "root", "1337superPass", provider)
		}

		time.Sleep(8 * time.Second)
		controller.CreateImage(token, provider, boxID, "Fleex-build-"+timeNow)
		if deleteFlag {
			time.Sleep(5 * time.Second)
			controller.DeleteFleet(fleetName, token, provider)
		}
	},
}

func init() {
	home, _ := homedir.Dir()
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean)")
	buildCmd.Flags().StringP("file", "f", home+"/fleex/build/test.yaml", "Build file")
	buildCmd.Flags().StringP("region", "R", "", "Region")
	buildCmd.Flags().StringP("size", "S", "", "Size")
	buildCmd.Flags().BoolP("delete", "d", false, "Delete box after image creation")

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
