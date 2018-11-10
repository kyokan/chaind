package cmd

import (
	"github.com/spf13/cobra"
	"bufio"
	"os"
	"fmt"
	"github.com/kyokan/chaind/pkg/config"
	"github.com/spf13/viper"
	"strings"
	"path"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "installs chaind",
	RunE: func(cmd *cobra.Command, args []string) error {
		install()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func install() {
	fmt.Println("Welcome to the chaind interactive installer.")
	home := prompt(
		"Where do you want to store your chaind configuration files and database?",
		viper.GetString(config.FlagHome),
		"",
	)
	useTLSStr := prompt(
		"Do you want to encrypt RPC calls using TLS?",
		"no",
		"yes/no",
	)
	useTLS := useTLSStr == "yes"
	var certPath string
	if useTLS {
		certPath = prompt(
			"Where can chaind find your certificate file?",
			"",
			"",
		)
	}
	ethUrl := prompt(
		"At what path should chaind serve Ethereum JSON-RPC requests?",
		"eth",
		"",
	)

	fmt.Print("Creating home directory...")
	maybeBail(os.MkdirAll(home, os.ModeDir|os.ModePerm))
	fmt.Println(" Done.")

	fmt.Print("Writing config file...")
	viper.Set(config.FlagHome, home)
	viper.Set(config.FlagCertPath, certPath)
	viper.Set(config.FlagUseTLS, useTLS)
	viper.Set(config.FlagETHURL, ethUrl)
	viper.SetConfigFile(path.Join(home, config.DefaultConfigFile))
	maybeBail(viper.WriteConfig())
	fmt.Println(" Done.")
	fmt.Printf("You're all set! To start your node run chaind --home %s start.\n", home)
}

func prompt(text string, def string, choices string) string {
	choiceMap := make(map[string]bool)
	allowed := strings.Split(choices, "/")
	for _, choice := range allowed {
		choiceMap[choice] = true
	}

	var scan func() string
	scan = func() string {
		scanner := bufio.NewScanner(os.Stdin)
		if def == "" {
			fmt.Printf("%s", text)
		} else {
			if choices == "" {
				fmt.Printf("%s [%s]: ", text, def)
			} else {
				fmt.Printf("%s [%s] (default %s): ", text, choices, def)
			}
		}
		scanner.Scan()
		out := strings.TrimSpace(scanner.Text())
		if out == "" {
			out = def
		}

		if choices != "" && !choiceMap[out] {
			fmt.Println("Invalid choice, please try again")
			return scan()
		}

		return out
	}

	return scan()
}

func maybeBail(err error) {
	if err == nil {
		return
	}

	fmt.Printf(" Failed! Reason: %s\n", err)
	os.Exit(1)
}
