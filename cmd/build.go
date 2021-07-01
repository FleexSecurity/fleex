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
		var token, region, size, sshFingerprint, boxIP, image string
		var boxID int
		timeNow := strconv.FormatInt(time.Now().Unix(), 10)
		home, _ := homedir.Dir()
		fleetName := "fleex-" + timeNow

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
		providerFlag = viper.GetString("provider")

		if regionFlag != "" {
			viper.Set(providerFlag+".region", regionFlag)
		}
		if sizeFlag != "" {
			viper.Set(providerFlag+".size", sizeFlag)
		}

		switch provider {
		case controller.PROVIDER_LINODE:
			token = viper.GetString("linode.token")
			region = viper.GetString("linode.region")
			size = viper.GetString("linode.size")
			image = "linode/ubuntu20.04"
		case controller.PROVIDER_DIGITALOCEAN:
			token = viper.GetString("digitalocean.token")
			region = viper.GetString("digitalocean.region")
			size = viper.GetString("digitalocean.size")
			sshFingerprint = sshutils.SSHFingerprintGen(publicSSH)
			image = "ubuntu-20-04-x64"
		}

		c, err := readConf(fileFlag)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(fileFlag)
		log.Fatal(1)
		controller.SpawnFleet(fleetName, 1, image, region, size, sshFingerprint, tags, token, false, provider)

		for {
			stillNotReady := false
			fleets := controller.GetFleet(fleetName+"-1", token, provider)
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
			log.Fatal(err)
		}

		for _, command := range c.Commands {
			controller.RunCommand(fleetName+"-1", command, token, 22, "root", "1337superPass", provider)
		}

		time.Sleep(8 * time.Second)
		controller.CreateImage(token, provider, boxID, "Fleex-build-"+timeNow)
		if deleteFlag {
			time.Sleep(5 * time.Second)
			controller.DeleteFleet(fleetName+"-1", token, provider)
		}
	},
}

func init() {
	home, _ := homedir.Dir()
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringP("provider", "p", "", "Service provider (Supported: linode, digitalocean)")
	buildCmd.Flags().StringP("file", "f", home+"/fleex/build/common.yaml", "Build file")
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
