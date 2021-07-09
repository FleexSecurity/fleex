package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sw33tLie/fleex/pkg/utils"
)

// initCmd represents the init command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "fleex config setup",
	Long:  "fleex config setup",
}

var configInit = &cobra.Command{
	Use:   "init",
	Short: "fleex init project",
	Long:  "fleex init project",
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
			fileUrl = "https://github.com/sw33tLie/fleex/releases/download/v1.0/config.zip"
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

		viper.SetConfigType("yaml")
		viper.ReadInConfig()
		pubSSH := viper.GetString("public-ssh-file")
		if pubSSH == "" {
			utils.Log.Fatal("You need to create a Key Pair for SSH")
		}

		utils.Copy(home+"/.ssh/"+pubSSH, home+"/fleex/configs/authorized_keys")

		utils.Log.Info("Init completed, your config files are in ~/fleex/")
	},
}

var configGet = &cobra.Command{
	Use:   "get",
	Short: "fleex get data from config file",
	Long:  "fleex get data from config file",
	Run: func(cmd *cobra.Command, args []string) {
		fieldFlag, _ := cmd.Flags().GetString("field")
		viper.SetConfigType("yaml")
		viper.ReadInConfig()

		if strings.Contains(fieldFlag, ",") {
			fields := strings.Split(fieldFlag, ",")
			for _, singleField := range fields {
				field := viper.Get(singleField)
				fmt.Println("-", singleField, ":", field)
			}
		} else {
			fmt.Println("-", fieldFlag, ":", viper.Get(fieldFlag))
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInit)
	configCmd.AddCommand(configGet)

	configInit.Flags().StringP("url", "u", "", "Config folder url")
	configInit.Flags().BoolP("overwrite", "o", false, "If the fleex folder exists overwrite it")
	configGet.Flags().StringP("field", "f", "", "field to retrieve, comma separated")
}
