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
	"strings"

	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/hnakamur/go-scp"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// scpCmd represents the scp command
var scpCmd = &cobra.Command{
	Use:   "scp",
	Short: "SCP client",
	Long:  "SCP client",
	Run: func(cmd *cobra.Command, args []string) {
		var token string

		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		usernameFlag, _ := cmd.Flags().GetString("username")
		passwordFlag, _ := cmd.Flags().GetString("password")
		sourceFlag, _ := cmd.Flags().GetString("source")
		portFlag, _ := cmd.Flags().GetInt("port")
		destinationFlag, _ := cmd.Flags().GetString("destination")
		nameFlag, _ := cmd.Flags().GetString("name")

		home, _ := homedir.Dir()

		if providerFlag != "" {
			viper.Set("provider", providerFlag)
		}
		provider := controller.GetProvider(viper.GetString("provider"))
		providerFlag = viper.GetString("provider")

		if usernameFlag != "" {
			viper.Set(providerFlag+".username", usernameFlag)
		}
		if passwordFlag != "" {
			viper.Set(providerFlag+".password", passwordFlag)
		}
		if portFlag != -1 {
			viper.Set(providerFlag+".port", portFlag)
		}

		switch provider {
		case controller.PROVIDER_LINODE:
			token = viper.GetString("linode.token")
			usernameFlag = viper.GetString("linode.username")
			passwordFlag = viper.GetString("linode.password")
			portFlag = viper.GetInt("linode.port")
		case controller.PROVIDER_DIGITALOCEAN:
			token = viper.GetString("digitalocean.token")
			usernameFlag = viper.GetString("digitalocean.username")
			passwordFlag = viper.GetString("digitalocean.password")
			portFlag = viper.GetInt("digitalocean.port")
		}

		if strings.Contains(destinationFlag, home) {
			if home != "/root" {
				destinationFlag = strings.ReplaceAll(destinationFlag, home, "/home/"+usernameFlag)
			}
		}

		fleets := controller.GetFleet(nameFlag, token, provider)
		if len(fleets) == 0 {
			utils.Log.Fatal("Box not found")
		}
		for _, box := range fleets {
			if box.Label == nameFlag {
				SendSCP(sourceFlag, destinationFlag, box.IP, portFlag, usernameFlag, passwordFlag)
				return
			}
		}

		for _, box := range fleets {
			if strings.HasPrefix(box.Label, nameFlag) {
				SendSCP(sourceFlag, destinationFlag, box.IP, portFlag, usernameFlag, passwordFlag)
			}
		}

		utils.Log.Info("SCP completed, you can find your files in " + destinationFlag)
	},
}

func init() {
	rootCmd.AddCommand(scpCmd)

	scpCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean)")
	scpCmd.Flags().StringP("name", "n", "pwn", "Fleet name")
	scpCmd.Flags().StringP("username", "U", "", "Username")
	scpCmd.Flags().StringP("password", "P", "", "Password")
	scpCmd.Flags().IntP("port", "", -1, "SSH port")
	scpCmd.Flags().StringP("source", "s", "", "Source file / folder")
	scpCmd.Flags().StringP("destination", "d", "", "Destination file / folder")

	scpCmd.MarkFlagRequired("source")
	scpCmd.MarkFlagRequired("destination")

}

func SendSCP(source string, destination string, IP string, PORT int, username string, password string) {
	checkDir, err := IsDirectory(source)
	if err != nil {
		utils.Log.Fatal(err)
	}
	if checkDir {
		err := scp.NewSCP(sshutils.GetConnection(IP, PORT, username, password).Client).SendDir(source, destination, nil)
		if err != nil {
			utils.Log.Fatal(err)
		}
	} else {
		err := scp.NewSCP(sshutils.GetConnection(IP, PORT, username, password).Client).SendFile(source, destination)
		if err != nil {
			utils.Log.Fatal(err)
		}
	}
}

func IsDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), err
}
