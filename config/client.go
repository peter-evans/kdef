package config

import (
	"github.com/peter-evans/kdef/core/client"
)

// Configuration options
type Options struct {
	ConfigPath string
	ConfigOpts []string
}

// Load config and create a new client
func NewClient(opts *Options) (*client.Client, error) {
	// Load config
	cc, err := loadConfig(opts.ConfigPath, opts.ConfigOpts)
	if err != nil {
		return nil, err
	}
	// Build client
	cl, err := client.New(cc)
	if err != nil {
		return nil, err
	}

	return cl, nil
}
