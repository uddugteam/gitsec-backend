package config

// Scheme is application config Scheme
type Scheme struct {
	// Application environment
	Env string

	// Application HTTP server
	Http *Http

	Git *Git
}

// Http is HTTP server config scheme
type Http struct {
	Port int
}

type Git struct {
	Path string
}
