package config

// Scheme is application config Scheme
type Scheme struct {
	// Application environment
	Env string

	// Application HTTP server
	Http *Http
}

// Http is HTTP server config scheme
type Http struct {
	Port int
}
