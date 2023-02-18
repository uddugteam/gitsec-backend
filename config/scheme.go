package config

// Scheme represents the application configuration scheme.
type Scheme struct {
	// Env is the application environment.
	Env string

	// HTTP is the configuration for the application HTTP server.
	HTTP *HTTP

	// Git is the configuration for the Git server.
	Git *Git

	// Ipfs is the configuration for the Ipfs client.
	Ipfs *Ipfs

	Blockchain *Blockchain

	// ETH account private key that will be using to sign outcoming transactions
	Signer string

	Baseurl string
}

// HTTP represents the HTTP server configuration scheme.
type HTTP struct {
	// Port is the port that the HTTP server should listen on.
	Port int
}

// Git represents the Git server configuration scheme.
type Git struct {
	// Path is the path to the Git repositories.
	Path string
}

// Ipfs represent Ipfs client configuration scheme.
type Ipfs struct {
	// Address of Ipfs node
	Address string
}

type Blockchain struct {
	Name     string
	Network  string
	Rpc      string
	Contract string
}
