package client

// Client configuration
type ClientConfig struct {
	SeedBrokers []string    `json:"seedBrokers,omitempty"`
	TLS         *tlsConfig  `json:"tls,omitempty"`
	SASL        *saslConfig `json:"sasl,omitempty"`

	// Set the maximum Kafka version to try (e.g. '0.8.0', '2.3.0')
	AsVersion string `json:"asVersion,omitempty"`
	// Underlying Kafka client log-level (none, error, warn, info, debug)
	LogLevel string `json:"logLevel,omitempty"`

	// The following configurations are not used to build client options, but instead
	// are exposed as methods on the Client to be used when making requests.

	// Timeout in milliseconds to be used by requests with timeouts
	TimeoutMs int32 `json:"timeoutMs,omitempty"`
	// The alter configs method that should be used (auto, incremental, non-incremental)
	AlterConfigsMethod string `json:"alterConfigsMethod,omitempty"`
}

// TLS configuration
type tlsConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	CACertPath     string `json:"caCertPath,omitempty"`
	ClientCertPath string `json:"clientCertPath,omitempty"`
	ClientKeyPath  string `json:"clientKeyPath,omitempty"`
	ServerName     string `json:"serverName,omitempty"`

	MinVersion       string   `json:"minVersion,omitempty"`
	CipherSuites     []string `json:"cipherSuites,omitempty"`
	CurvePreferences []string `json:"curvePreferences,omitempty"`
}

// SASL configuration
type saslConfig struct {
	Method  string `json:"method,omitempty"`
	Zid     string `json:"zid,omitempty"`
	User    string `json:"user,omitempty"`
	Pass    string `json:"pass,omitempty"`
	IsToken bool   `json:"isToken,omitempty"`
}

// Valid values for configuring the alter configs method
var alterConfigsMethodValidValues = []string{"auto", "incremental", "non-incremental"}
