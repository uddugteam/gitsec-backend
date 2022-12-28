package config

// Scheme represents the application configuration scheme.
type Scheme struct {
	// Env is the application environment.
	Env string

	// Http is the configuration for the application HTTP server.
	Http *Http

	// Git is the configuration for the Git server.
	Git *Git
}

// Http represents the HTTP server configuration scheme.
type Http struct {
	// Port is the port that the HTTP server should listen on.
	Port int
}

// Git represents the Git server configuration scheme.
type Git struct {
	// Path is the path to the Git repositories.
	Path string
}
