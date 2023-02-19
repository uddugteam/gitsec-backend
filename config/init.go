package config

import (
	"github.com/spf13/viper"
)

// init initialize default config params
func init() {
	// environment - could be "local", "prod", "dev"
	viper.SetDefault("env", "prod")

	viper.SetDefault("baseurl", "http://localhost:8080/")

	// http server
	viper.SetDefault("http.port", 8080)

	viper.SetDefault("git.path", ".repos/")

	viper.SetDefault("ipfs.address", "http://127.0.0.1:5001")

	viper.SetDefault("blockchain.name", "gnosis")
	viper.SetDefault("blockchain.network", "chiado")
	viper.SetDefault("blockchain.rpc", "wss://rpc.chiado.gnosis.gateway.fm/ws")
	viper.SetDefault("blockchain.contract", "")

	// signer private key
	viper.SetDefault("signer", "")

	viper.SetDefault("pinata.jwt", "")

	viper.SetDefault("pinner", "pinata")
}
