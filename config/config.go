package config

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
	"github.com/peter-evans/kdef/core/client"
)

const (
	defaultConfigFilename = "config.yml"
	envVarPrefix          = "KDEF__"
)

// Default client configuration
var defaultClientConfig = map[string]interface{}{
	"seedBrokers":        []string{"localhost:9092"},
	"timeoutMs":          5000,
	"tls.enabled":        false,
	"asVersion":          "",
	"logLevel":           "none",
	"alterConfigsMethod": "auto",
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
func loadConfig(configPath string, configOpts []string) (*client.ClientConfig, error) {
	log.Debug("Loading client config")

	var k = koanf.New(".")

	// Load default values
	if err := k.Load(confmap.Provider(defaultClientConfig, "."), nil); err != nil {
		return nil, err
	}

	// Load config file
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		if os.IsNotExist(err) {
			log.Debug("No config file found at path %q", configPath)
		} else {
			return nil, fmt.Errorf("failed to load config file %q: %v", configPath, err)
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
	if err := k.Load(env.ProviderWithValue(envVarPrefix, ".", func(s string, v string) (string, interface{}) {
		// Trim the prefix, lowercase, and replace "__" with the "." key delimiter
		key := strings.Replace(strings.ToLower(strings.TrimPrefix(s, envVarPrefix)), "__", ".", -1)
		// Convert to camelcase, e.g. "seed_brokers" -> "seedBrokers"
		key = strcase.ToLowerCamel(key)

		return key, typedVal(key, v)
	}), nil); err != nil {
		return nil, err
	}

	// Load commandline flag overrides
	flagConfigOptsMap := map[string]interface{}{}
	for _, opt := range configOpts {
		kv := strings.SplitN(opt, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("config option %q not a 'key=value' pair", opt)
		}
		flagConfigOptsMap[kv[0]] = typedVal(kv[0], kv[1])
	}
	if err := k.Load(confmap.Provider(flagConfigOptsMap, "."), nil); err != nil {
		return nil, err
	}

	// Unmarshal to config struct
	cc := &client.ClientConfig{}
	if err := k.UnmarshalWithConf("", cc, koanf.UnmarshalConf{Tag: "json"}); err != nil {
		return nil, err
	}

	for _, key := range k.Keys() {
		log.Debug("%s: %v", key, k.Get(key))
	}

	return cc, nil
}
