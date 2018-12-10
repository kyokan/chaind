package config

import (
	"github.com/spf13/viper"
	"path"
	"os"
	"log"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"errors"
	"github.com/kyokan/chaind/pkg"
	"net/url"
	"github.com/kyokan/chaind/pkg/sets"
)

const DefaultHome = "~/.chaind"
const DefaultConfigFile = "chaind.toml"

const (
	FlagHome     = "home"
	FlagCertPath = "cert_path"
	FlagUseTLS   = "use_tls"
)

var ValidETHAPIs = sets.NewStringSet([]string{
	"admin",
	"db",
	"debug",
	"eth",
	"miner",
	"net",
	"personal",
	"shh",
	"txpool",
	"web3",
})

type Config struct {
	Home             string            `mapstructure:"home"`
	CertPath         string            `mapstructure:"cert_path"`
	UseTLS           bool              `mapstructure:"use_tls"`
	EnablePrometheus bool              `mapstructure:"enable_prometheus"`
	ETHConfig        *ETH              `mapstructure:"eth"`
	RPCPort          int               `mapstructure:"rpc_port"`
	LogLevel         string            `mapstructure:"log_level"`
	LogAuditorConfig *LogAuditorConfig `mapstructure:"log_auditor"`
	RedisConfig      *RedisConfig      `mapstructure:"redis"`
	Backends         []Backend         `mapstructure:"backend"`
	Master           bool              `mapstructure:"master"`
}

type LogAuditorConfig struct {
	LogFile string `mapstructure:"log_file"`
}

type RedisConfig struct {
	URL      string `mapstructure:"url"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type Backend struct {
	Type pkg.BackendType `mapstructure:"type"`
	URL  string          `mapstructure:"url"`
	Name string          `mapstructure:"name"`
	Main bool            `mapstructure:"main"`
}

type ETH struct {
	APIs []string `mapstructure:"apis"`
	Path string   `mapstructure:"path"`
}

func init() {
	home := mustExpand(DefaultHome)
	port := os.Getenv("PORT")
	rmt.Println("port -", port)
	if port == "" {
        log.Fatal("$PORT must be set")
	}
	
	viper.SetDefault(FlagHome, home)
	viper.SetDefault(FlagCertPath, "")
	viper.SetDefault(FlagUseTLS, false)
	viper.SetDefault(port, 8080)
}

func ReadConfig(allowDefaults bool) (Config, error) {
	var cfg Config
	cfgFile := path.Join(viper.GetString(FlagHome), DefaultConfigFile)
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		if allowDefaults {
			viper.Unmarshal(&cfg)
			return cfg, nil
		} else {
			return cfg, errors.New("config file not found")
		}
	}

	viper.SetConfigFile(cfgFile)
	if err := viper.ReadInConfig(); err != nil {
		return cfg, err
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, err
	}
	viper.Set(FlagHome, mustExpand(viper.GetString(FlagHome)))
	viper.Set(FlagCertPath, mustExpand(viper.GetString(FlagCertPath)))

	return cfg, nil
}

func ValidateConfig(cfg *Config) error {
	if len(cfg.Backends) == 0 {
		return validationError("must define at least one backend")
	}

	var hasMainBackend bool
	for _, backend := range cfg.Backends {
		if backend.Main && hasMainBackend {
			return validationError("cannot have more than one main backend")
		} else if backend.Main {
			hasMainBackend = true
		}

		if backend.Type != pkg.EthBackend {
			return validationError("only Ethereum backends are supported right now")
		}

		_, err := url.Parse(backend.URL)
		if err != nil {
			return validationError(fmt.Sprintf("invalid url: %s", backend.URL))
		}

		if backend.Name == "" {
			return validationError("backend name must be defined")
		}
	}

	if cfg.ETHConfig != nil {
		if !ValidETHAPIs.ContainsAll(cfg.ETHConfig.APIs) {
			return validationError("invalid API provided")
		}
	}

	return nil
}

func validationError(msg string) error {
	return errors.New(fmt.Sprintf("invalid config: %s", msg))
}

func mustExpand(path string) string {
	expanded, err := homedir.Expand(path)
	if err != nil {
		fmt.Println("Failed to find home directory on this system. Exiting.")
		os.Exit(1)
	}

	return expanded
}
