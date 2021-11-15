// Package config implements loading config from several sources and client creation.
package config

import (
	"github.com/peter-evans/kdef/core/client"
)

// Options represent client configuration options.
type Options struct {
	ConfigPath string
	ConfigOpts []string
}

// NewClient loads configuration from several sources and creates a new client.
func NewClient(opts *Options) (*client.Client, error) {
	cc, err := loadConfig(opts.ConfigPath, opts.ConfigOpts)
	if err != nil {
		return nil, err
	}

	cl, err := client.New(cc)
	if err != nil {
		return nil, err
	}

	return cl, nil
}
