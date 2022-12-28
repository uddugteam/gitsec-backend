package config

import (
	"github.com/spf13/viper"
)

// init initialize default config params
func init() {
	// environment - could be "local", "prod", "dev"
	viper.SetDefault("env", "prod")

	// http server
	viper.SetDefault("http.port", 8080)

	viper.SetDefault("git.path", ".repos/")
}
