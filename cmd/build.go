package cmd

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/hnakamur/go-scp"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
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
	Short: "Build an image with all the tools you need. Run this the first time only (for each provider).",
	Long:  "Build image",
	Run: func(cmd *cobra.Command, args []string) {
		var token, region, size, boxIP, image string
		var boxID string

		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		providerFlag, _ := cmd.Flags().GetString("provider")
		regionFlag, _ := cmd.Flags().GetString("region")
		sizeFlag, _ := cmd.Flags().GetString("size")
		fileFlag, _ := cmd.Flags().GetString("file")
		noDeleteFlag, _ := cmd.Flags().GetBool("no-delete")
		debugFlag, _ := cmd.Flags().GetBool("debug")

		timeNow := strconv.FormatInt(time.Now().Unix(), 10)
		home, _ := homedir.Dir()
		fleetName := "fleex-" + timeNow

		pubSSH := globalConfig.SSHKeys.PublicFile
		if pubSSH == "" {
			utils.Log.Fatal("You need to create a Key Pair for SSH")
		}

		providerInfo := globalConfig.Providers[providerFlag]
		providerInfo.Tags = []string{"snapshot"}
		globalConfig.Providers[providerFlag] = providerInfo

		if globalConfig.Settings.Provider != providerFlag && providerFlag == "" {
			providerFlag = globalConfig.Settings.Provider
		}

		provider := controller.GetProvider(providerFlag)
		if provider == -1 {
			utils.Log.Fatal(models.ErrInvalidProvider)
		}
		token = globalConfig.Providers[providerFlag].Token

		if regionFlag == "" {
			regionFlag = globalConfig.Providers[providerFlag].Region
		}
		if sizeFlag == "" {
			sizeFlag = globalConfig.Providers[providerFlag].Size

		}
		token = globalConfig.Providers[providerFlag].Token
		switch provider {
		case controller.PROVIDER_LINODE:
			image = "linode/ubuntu20.04"
		case controller.PROVIDER_DIGITALOCEAN:
			image = "ubuntu-20-04-x64"
		case controller.PROVIDER_VULTR:
			image = "270"
		}

		destinationPath := filepath.Join(home, "fleex/configs/authorized_keys")

		sourcePath := filepath.Join(home, ".ssh", pubSSH)
		utils.Copy(sourcePath, destinationPath)

		newController := controller.NewController(globalConfig)

		if provider == controller.PROVIDER_LINODE {
			packerVars := "-var 'TOKEN=" + token + "'"
			packerVars += " -var 'IMAGE=" + image + "'"
			packerVars += " -var 'SIZE=" + size + "'"
			packerVars += " -var 'REGION=" + region + "'"
			utils.RunCommand("packer build "+packerVars+" "+fileFlag, debugFlag)
		} else {
			c, err := readConf(fileFlag)
			if err != nil {
				utils.Log.Fatal(err)
			}

			newController.SpawnFleet(fleetName, 1, false, true)

			for {
				stillNotReady := false
				fleets := newController.GetFleet(fleetName + "-1")
				if len(fleets) == 0 {
					stillNotReady = true
				}
				for _, box := range fleets {
					if box.Label == fleetName+"-1" {
						boxID = box.ID
						boxIP = box.IP
						break
					}
				}

				if stillNotReady {
					time.Sleep(3 * time.Second)
				} else {
					break
				}
			}

			if strings.ContainsAny("~", c.Config.Source) {
				c.Config.Source = strings.ReplaceAll(c.Config.Source, "~", home)
			}

			for {
				stillNotReady := false
				_, err := sshutils.GetConnectionBuild(boxIP, 22, "root", "1337superPass")
				if err != nil {
					stillNotReady = true
				}

				if stillNotReady {
					time.Sleep(5 * time.Second)
				} else {
					break
				}
			}

			err = scp.NewSCP(sshutils.GetConnection(boxIP, 22, "root", "1337superPass").Client).SendDir(c.Config.Source, c.Config.Destination, nil)
			if err != nil {
				utils.Log.Fatal(err)
			}

			if provider == controller.PROVIDER_DIGITALOCEAN {
				c.Commands = append(c.Commands, `/bin/su -l op -c "curl http://169.254.169.254/metadata/v1/user-data > /home/op/install.sh"`)
				c.Commands = append(c.Commands, `/bin/su -l op -c "chmod +x /home/op/install.sh"`)
				c.Commands = append(c.Commands, `/bin/su -l op -c "/home/op/install.sh"`)
			}

			for _, command := range c.Commands {
				prov := newController.Configs.Settings.Provider
				provInfo := newController.Configs.Providers[prov]
				provInfo.Port = 22
				provInfo.Token = token
				provInfo.Username = "root"
				provInfo.Password = "1337superPass"
				newController.RunCommand(fleetName+"-1", command)
			}

			time.Sleep(8 * time.Second)
			newController.CreateImage(token, provider, boxID, "Fleex-build-"+timeNow)
			if !noDeleteFlag {
				time.Sleep(5 * time.Second)
				newController.DeleteFleet(fleetName + "-1")
			}
			utils.Log.Info("\nImage done!")
		}
	},
}

func init() {
	home, _ := homedir.Dir()
	// rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean, vultr)")
	buildCmd.Flags().StringP("file", "f", home+"/fleex/build/common.yaml", "Build file")
	buildCmd.Flags().StringP("region", "R", "", "Region")
	buildCmd.Flags().StringP("size", "S", "", "Size")
	buildCmd.Flags().BoolP("no-delete", "", false, "Don't delete the box after image creation")
	buildCmd.Flags().BoolP("debug", "D", false, "Show build logs")

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
