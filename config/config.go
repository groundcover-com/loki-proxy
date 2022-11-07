package config

import (
	_ "embed"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

//go:embed config.yaml
var defaultConfig string

type Config struct {
	Target struct {
		Url       string
		TenantId  string `mapstructure:"tenant_id"`
		LabelName string `mapstructure:"label_name"`
	}
	Bind struct {
		Port    int
		Address string
	}
}

func NewConfig() (*Config, error) {
	var config Config
	var err error

	viper.AutomaticEnv()
	// these will resolve to config/config.yaml
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath("config")
	// end
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.ReadConfig(strings.NewReader(defaultConfig))

	if err = viper.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (config *Config) Print() {
	fmt.Printf("config: %+v\n", config)
}

func (config *Config) BindAddr() string {
	return fmt.Sprintf("%s:%d", config.Bind.Address, config.Bind.Port)
}

func (config *Config) TargetUrl() (*url.URL, error) {
	var err error

	var targetUrl *url.URL
	if targetUrl, err = url.Parse(config.Target.Url); err != nil {
		return nil, err
	}

	return targetUrl, nil
}
