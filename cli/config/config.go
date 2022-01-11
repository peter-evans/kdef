// Package config implements loading config from several sources and client creation.
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
	"github.com/peter-evans/kdef/core/util/str"
)

const (
	defaultConfigFilename = "config.yml"
	envVarPrefix          = "KDEF__"
)

var defaultClientConfig = map[string]interface{}{
	"seedBrokers":        []string{"localhost:9092"},
	"timeoutMs":          5000,
	"tls.enabled":        false,
	"asVersion":          "",
	"logLevel":           "none",
	"alterConfigsMethod": "auto",
}

var sensitiveConfigKeys = []string{
	"sasl.pass",
}

// DefaultConfigPath determines the default configuration file path.
func DefaultConfigPath() string {
	dir, _ := os.Getwd()
	if d, ok := os.LookupEnv("KDEF_CONFIG_DIR"); ok {
		dir = d
	}
	filename := defaultConfigFilename
	if f, ok := os.LookupEnv("KDEF_CONFIG_FILENAME"); ok {
		filename = f
	}
	path := filepath.Join(dir, filename)
	if p, ok := os.LookupEnv("KDEF_CONFIG_PATH"); ok {
		path = p
	}

	return path
}

func loadConfig(configPath string, configOpts []string) (*client.Config, error) {
	log.Debugf("Loading client config")

	k := koanf.New(".")

	// Load defaults
	if err := k.Load(confmap.Provider(defaultClientConfig, "."), nil); err != nil {
		return nil, err
	}

	// Load config file
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		if os.IsNotExist(err) {
			log.Debugf("No config file found at path %q", configPath)
		} else {
			return nil, fmt.Errorf("failed to load config file %q: %v", configPath, err)
		}
	}

	// typedVal converts a key's value to its correct type
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
		key := strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, envVarPrefix)), "__", ".")
		// Convert to camelcase, e.g. "seed_brokers" -> "seedBrokers"
		key = strcase.ToLowerCamel(key)
		return key, typedVal(key, v)
	}), nil); err != nil {
		return nil, err
	}

	// Load commandline option overrides
	optsMap := map[string]interface{}{}
	for _, opt := range configOpts {
		kv := strings.SplitN(opt, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("config option %q not a 'key=value' pair", opt)
		}
		optsMap[kv[0]] = typedVal(kv[0], kv[1])
	}
	if err := k.Load(confmap.Provider(optsMap, "."), nil); err != nil {
		return nil, err
	}

	cc := &client.Config{}
	if err := k.UnmarshalWithConf("", cc, koanf.UnmarshalConf{Tag: "json"}); err != nil {
		return nil, err
	}

	for _, key := range k.Keys() {
		var val interface{} = "***"
		if !str.Contains(key, sensitiveConfigKeys) {
			val = k.Get(key)
		}
		log.Debugf("%s: %v", key, val)
	}

	return cc, nil
}
