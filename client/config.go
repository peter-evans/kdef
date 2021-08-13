package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/peter-evans/kdef/cli/log"
)

// Client configuration
type clientConfig struct {
	SeedBrokers []string    `json:"seedBrokers,omitempty"`
	TimeoutMs   int32       `json:"timeoutMs,omitempty"`
	TLS         *tlsConfig  `json:"tls,omitempty"`
	SASL        *saslConfig `json:"sasl,omitempty"`

	// Set the maximum Kafka version to try (e.g. '0.8.0', '2.3.0')
	AsVersion string `json:"asVersion,omitempty"`
	// Underlying Kafka client log-level (none, error, warn, info, debug)
	LogLevel string `json:"logLevel,omitempty"`
}

// TLS configuration
type tlsConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	CACertPath     string `json:"caCertPath,omitempty"`
	ClientCertPath string `json:"clientCertPath,omitempty"`
	ClientKeyPath  string `json:"clientKeyPath,omitempty"`
	ServerName     string `json:"serverName,omitempty"`

	MinVersion       string   `json:"minVersion,omitempty"`
	CipherSuites     []string `json:"cipherSuites"`
	CurvePreferences []string `json:"curvePreferences"`
}

// SASL configuration
type saslConfig struct {
	Method  string `json:"method,omitempty"`
	Zid     string `json:"zid,omitempty"`
	User    string `json:"user,omitempty"`
	Pass    string `json:"pass,omitempty"`
	IsToken bool   `json:"isToken,omitempty"`
}

const (
	defaultConfigFilename = "config.yml"
	envVarPrefix          = "KDEF__"
)

// Default client configuration
var defaultClientConfig = map[string]interface{}{
	"seedBrokers": []string{"localhost:9092"},
	"timeoutMs":   5000,
	"tls.enabled": false,
	"asVersion":   "",
	"logLevel":    "none",
}

// The default config file path
func DefaultConfigPath() string {
	directory, _ := os.Getwd()
	if d, ok := os.LookupEnv("KDEF_CONFIG_DIR"); ok {
		directory = d
	}
	filename := defaultConfigFilename
	if f, ok := os.LookupEnv("KDEF_CONFIG_FILENAME"); ok {
		filename = f
	}
	path := filepath.Join(directory, filename)
	if p, ok := os.LookupEnv("KDEF_CONFIG_PATH"); ok {
		path = p
	}

	return path
}

// Loads and merges the client configuration from multiple sources
func loadConfig(filepath string, configOpts []string) (*clientConfig, error) {
	log.Debug("Loading client config")

	var k = koanf.New(".")

	// Load default values
	k.Load(confmap.Provider(defaultClientConfig, "."), nil)

	// Load config file
	if err := k.Load(file.Provider(filepath), yaml.Parser()); err != nil {
		if os.IsNotExist(err) {
			log.Debug("No config file found at path %q", filepath)
		} else {
			return nil, fmt.Errorf("failed to load config file %q: %v", filepath, err)
		}
	}

	// Convert a key's value to its correct type
	typedVal := func(k string, v string) interface{} {
		switch k {
		case "seedBrokers":
			return strings.Split(v, ",")
		default:
			return v
		}
	}

	// Load environment variable overrides
	k.Load(env.ProviderWithValue(envVarPrefix, ".", func(s string, v string) (string, interface{}) {
		// Trim the prefix, lowercase, and replace "__" with the "." key delimiter
		key := strings.Replace(strings.ToLower(strings.TrimPrefix(s, envVarPrefix)), "__", ".", -1)
		// Convert to camelcase, e.g. "seed_brokers" -> "seedBrokers"
		key = strcase.ToLowerCamel(key)

		return key, typedVal(key, v)
	}), nil)

	// Load commandline flag overrides
	flagConfigOptsMap := map[string]interface{}{}
	for _, opt := range configOpts {
		kv := strings.SplitN(opt, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("config option %q not a 'key=value' pair", opt)
		}
		flagConfigOptsMap[kv[0]] = typedVal(kv[0], kv[1])
	}
	k.Load(confmap.Provider(flagConfigOptsMap, "."), nil)

	// Unmarshal to config struct
	cc := &clientConfig{}
	k.UnmarshalWithConf("", cc, koanf.UnmarshalConf{Tag: "json"})

	for _, key := range k.Keys() {
		log.Debug("%s: %v", key, k.Get(key))
	}

	return cc, nil
}
