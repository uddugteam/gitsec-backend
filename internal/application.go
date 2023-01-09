package internal

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/misnaged/annales/logger"
	version "github.com/misnaged/annales/versioner"

	"gitsec-backend/config"
	"gitsec-backend/internal/server"
	"gitsec-backend/internal/service"
)

// App is main microservice application instance that
// have all necessary dependencies inside structure
type App struct {
	// application configuration
	config *config.Scheme

	version *version.Version

	httpServer *server.HTTPServer

	srv service.IGitService
}

// NewApplication create new App instance
func NewApplication() (app *App, err error) {
	ver, err := version.NewVersion()
	if err != nil {
		return nil, fmt.Errorf("init app version: %w", err)
	}

	return &App{
		config:  &config.Scheme{},
		version: ver,
	}, nil
}

// Init initialize application and all necessary instances
func (app *App) Init() (err error) {
	app.srv, err = service.NewGitService(app.Config().Git)
	if err != nil {
		return fmt.Errorf("initialize application service layer: %w", err)
	}

	app.httpServer = server.NewHTTPServer(app.Config(), app.srv)

	return nil
}

// Serve start serving Application service
func (app *App) Serve() error {
	go func() {
		logger.Log().Info(fmt.Sprintf("Listen HTTP Server on :%d", app.config.HTTP.Port))

		if err := app.httpServer.ListenAndServe(); err != nil {
			logger.Log().Fatal(err)
		}
	}()

	// Gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-quit

	if err := app.Stop(); err != nil {
		return fmt.Errorf("error by stopping app: %w", err)
	}
	return nil
}

// Stop shutdown the application
func (app *App) Stop() error {
	if err := app.httpServer.Close(); err != nil {
		return fmt.Errorf("close httpServer listening: %w", err)
	}

	return nil
}

// Config return App config Scheme
func (app *App) Config() *config.Scheme {
	return app.config
}

// Version return application current version
func (app *App) Version() string {
	return app.version.String()
}

// CreateAddr is created address string from host and port
func CreateAddr(host string, port int) string {
	return fmt.Sprintf("%s:%v", host, port)
}
