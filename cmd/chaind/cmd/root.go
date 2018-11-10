package cmd

import (
	"github.com/spf13/cobra"
	"fmt"
	"os"
	"github.com/kyokan/chaind/pkg/config"
	"github.com/spf13/viper"
)

var home string
var rootCmd = &cobra.Command{
	Use:   "chaind",
	Short: "a daemon that proxies and logs requests to blockchain nodes",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&home, config.FlagHome, "", "chaind home directory")
	viper.BindPFlag(config.FlagHome, rootCmd.PersistentFlags().Lookup(config.FlagHome))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
