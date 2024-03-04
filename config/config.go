package config

import (
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	// StaticDir stores the name of the directory where we will serve static files from
	StaticDir = "static"
	// StaticFiles stores the URL prefix used when serving static files
	StaticPrefix = "files"
	// TemplateExt stores the extension used for the template files
	TemplateExt = ".tmpl"
	// TemplateDir stores the name of the directory where we will store our templates
	TemplateDir = "../templates"
)

type (
	// Complete application configuration
	Config struct {
		App   AppConfig
		Cache CacheConfig
		HTTP  HTTPConfig
	}

	// Application configuration
	AppConfig struct {
		EncryptionKey string
		Environment   environment
		Name          string
		Timeout       time.Duration
	}

	CacheConfig struct {
		Database     int
		Hostname     string
		Password     string
		Port         uint16
		TestDatabase int
		Expiration   struct {
			Page       time.Duration
			StaticFile time.Duration
		}
	}

	// HTTP server configuration
	HTTPConfig struct {
		Hostname string
		Port     uint16
		TLS      struct {
			Enabled bool
		}
	}
)

func GetConfig() (Config, error) {
	var c Config

	// Set default values
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("config")
	viper.AddConfigPath("../config")
	viper.AddConfigPath("../../config")

	// Read config
	viper.SetEnvPrefix("tinyblob")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return c, err
	}

	if err := viper.Unmarshal(&c); err != nil {
		return c, err
	}

	return c, nil
}

type environment string

const (
	EnvTest  environment = "test"
	EnvLocal environment = "local"
	EnvDev   environment = "development"
	EnvProd  environment = "production"
)

func SwitchEnv(env environment) {
	if err := os.Setenv("TINYBLOB_APP_ENVIRONMENT", string(env)); err != nil {
		panic(err)
	}
}
