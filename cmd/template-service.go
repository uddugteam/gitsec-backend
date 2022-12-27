package main

import (
	"os"

	"github.com/Misnaged/annales/logger"

	"gitsec-backend/cmd/root"
	"gitsec-backend/cmd/serve"
	"gitsec-backend/internal"
)

func main() {
	app, err := internal.NewApplication()
	if err != nil {
		logger.Log().Infof("An error occurred: %s", err.Error())
		os.Exit(1)
	}

	rootCmd := root.Cmd(app)
	rootCmd.AddCommand(serve.Cmd(app))

	if err := rootCmd.Execute(); err != nil {
		logger.Log().Infof("An error occurred: %s", err.Error())
		os.Exit(1)
	}
}
