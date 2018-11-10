package cmd

import (
	"github.com/spf13/cobra"
	"github.com/kyokan/chaind/pkg/config"
	"github.com/kyokan/chaind/internal"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "starts chaind",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.ReadConfig(false)
		if err != nil {
			return err
		}

		return internal.Start(&cfg)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
