package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/util/str"

	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kversion"
	"github.com/twmb/franz-go/pkg/sasl/aws"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

// Create a new client
func New(cc *ClientConfig) (*Client, error) {
	cl := &Client{
		cc: cc,
		kgoOpts: []kgo.Opt{
			kgo.MetadataMinAge(time.Second),
		},
	}
	// Validate configuration not used for client options
	if err := cl.validateNonClientOptConfig(); err != nil {
		return nil, err
	}
	// Build the internal client
	if err := cl.buildClient(); err != nil {
		return nil, err
	}

	return cl, nil
}

// A client providing broker APIs
type Client struct {
	Client  *kgo.Client
	cc      *ClientConfig
	kgoOpts []kgo.Opt
}

// Timeout in milliseconds to be used by requests with timeouts
func (cl *Client) TimeoutMs() int32 {
	return cl.cc.TimeoutMs
}

// The alter configs method that should be used (auto, incremental, non-incremental)
func (cl *Client) AlterConfigsMethod() string {
	return cl.cc.AlterConfigsMethod
}

// Validate configuration not used for client options
func (cl *Client) validateNonClientOptConfig() error {
	if cl.cc.TimeoutMs < 0 {
		return fmt.Errorf("timeoutMs must be greater or equal to 0")
	}

	if !str.Contains(cl.cc.AlterConfigsMethod, alterConfigsMethodValidValues) {
		return fmt.Errorf("alterConfigsMethod must be one of %q", strings.Join(alterConfigsMethodValidValues, "|"))
	}

	return nil
}

// Build the internal client
func (cl *Client) buildClient() error {
	log.Debug("Building Kafka client")

	// Build options to configure a kgo.Client instance
	if err := cl.buildOptions(); err != nil {
		return err
	}

	// Create the client
	var err error
	cl.Client, err = kgo.NewClient(cl.kgoOpts...)
	if err != nil {
		return err
	}

	return nil
}

// Adds an option to configure kgo.Client
func (cl *Client) addOpt(opt kgo.Opt) {
	cl.kgoOpts = append(cl.kgoOpts, opt)
}

// Build options to configure a kgo.Client instance
func (cl *Client) buildOptions() error {
	// Set the seed brokers
	cl.addOpt(kgo.SeedBrokers(cl.cc.SeedBrokers...))

	// Build and set the SASL option
	if err := cl.buildSASLOpt(); err != nil {
		return err
	}

	// Build and set the TLS option
	if err := cl.buildTLSOpt(); err != nil {
		return err
	}

	// Build and set the max versions option
	if err := cl.buildMaxVersionsOpt(); err != nil {
		return err
	}

	// Build and set the client log level option
	if err := cl.buildLogLevelOpt(); err != nil {
		return err
	}

	return nil
}

// Build and set the SASL option
func (cl *Client) buildSASLOpt() error {
	if cl.cc.SASL == nil {
		return nil
	}

	switch str.Norm(cl.cc.SASL.Method) {
	case "", "plain":
		cl.addOpt(kgo.SASL(plain.Plain(func(context.Context) (plain.Auth, error) {
			return plain.Auth{
				Zid:  cl.cc.SASL.Zid,
				User: cl.cc.SASL.User,
				Pass: cl.cc.SASL.Pass,
			}, nil
		})))
	case "scramsha256":
		cl.addOpt(kgo.SASL(scram.Auth{
			Zid:     cl.cc.SASL.Zid,
			User:    cl.cc.SASL.User,
			Pass:    cl.cc.SASL.Pass,
			IsToken: cl.cc.SASL.IsToken,
		}.AsSha256Mechanism()))
	case "scramsha512":
		cl.addOpt(kgo.SASL(scram.Auth{
			Zid:     cl.cc.SASL.Zid,
			User:    cl.cc.SASL.User,
			Pass:    cl.cc.SASL.Pass,
			IsToken: cl.cc.SASL.IsToken,
		}.AsSha512Mechanism()))
	case "awsmskiam":
		sess, err := session.NewSession()
		if err != nil {
			return fmt.Errorf("failed to create aws session: %v", err)
		}
		cl.addOpt(kgo.SASL(aws.ManagedStreamingIAM(func(ctx context.Context) (aws.Auth, error) {
			creds, err := sess.Config.Credentials.GetWithContext(ctx)
			if err != nil {
				return aws.Auth{}, err
			}
			return aws.Auth{
				AccessKey:    creds.AccessKeyID,
				SecretKey:    creds.SecretAccessKey,
				SessionToken: creds.SessionToken,
			}, nil
		})))
	default:
		return fmt.Errorf("invalid sasl method %q", cl.cc.SASL.Method)
	}

	return nil
}

// Build and set the TLS option
func (cl *Client) buildTLSOpt() error {
	if !cl.cc.TLS.Enabled {
		return nil
	}

	tc := new(tls.Config)

	// Set min version
	switch strings.ToLower(cl.cc.TLS.MinVersion) {
	case "", "v1.2", "1.2":
		tc.MinVersion = tls.VersionTLS12 // Default
	case "v1.3", "1.3":
		tc.MinVersion = tls.VersionTLS13
	case "v1.1", "1.1":
		tc.MinVersion = tls.VersionTLS11
	case "v1.0", "1.0":
		tc.MinVersion = tls.VersionTLS10
	default:
		return fmt.Errorf("invalid tls min version %q", cl.cc.TLS.MinVersion)
	}

	// Set cipher suites
	if suites := cl.cc.TLS.CipherSuites; len(suites) > 0 {
		candidates := make(map[string]uint16)
		for _, suite := range append(tls.CipherSuites(), tls.InsecureCipherSuites()...) {
			candidates[str.Norm(suite.Name)] = suite.ID
			candidates[str.Norm(strings.TrimPrefix("TLS_", suite.Name))] = suite.ID
		}

		for _, suite := range cl.cc.TLS.CipherSuites {
			id, exists := candidates[str.Norm(suite)]
			if !exists {
				return fmt.Errorf("invalid cipher suite %q", suite)
			}
			tc.CipherSuites = append(tc.CipherSuites, id)
		}
	}

	// Set curve preferences
	if curves := cl.cc.TLS.CurvePreferences; len(curves) > 0 {
		candidates := map[string]tls.CurveID{
			"curvep256": tls.CurveP256,
			"curvep384": tls.CurveP384,
			"curvep521": tls.CurveP521,
			"x25519":    tls.X25519,
		}

		for _, curve := range cl.cc.TLS.CurvePreferences {
			id, exists := candidates[str.Norm(curve)]
			if !exists {
				return fmt.Errorf("invalid curve preference %q", curve)
			}
			tc.CurvePreferences = append(tc.CurvePreferences, id)
		}
	}

	// Set CA cert
	if len(cl.cc.TLS.CACertPath) > 0 {
		ca, err := ioutil.ReadFile(cl.cc.TLS.CACertPath)
		if err != nil {
			return fmt.Errorf("failed to read CA cert %q: %v", cl.cc.TLS.CACertPath, err)
		}

		tc.RootCAs = x509.NewCertPool()
		tc.RootCAs.AppendCertsFromPEM(ca)
	}

	// Set client cert
	if len(cl.cc.TLS.ClientCertPath) > 0 || len(cl.cc.TLS.ClientKeyPath) > 0 {
		if len(cl.cc.TLS.ClientCertPath) == 0 || len(cl.cc.TLS.ClientKeyPath) == 0 {
			return fmt.Errorf("both client and key cert paths must be provided, but only one found")
		}

		cert, err := ioutil.ReadFile(cl.cc.TLS.ClientCertPath)
		if err != nil {
			return fmt.Errorf("failed to read client cert %q: %v", cl.cc.TLS.ClientCertPath, err)
		}

		key, err := ioutil.ReadFile(cl.cc.TLS.ClientKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read client key %q: %v", cl.cc.TLS.ClientKeyPath, err)
		}

		pair, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return fmt.Errorf("failed to create key pair: %v", err)
		}
		tc.Certificates = append(tc.Certificates, pair)
	}

	// Add TLS opt
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	cl.addOpt(kgo.Dialer(func(_ context.Context, _, host string) (net.Conn, error) {
		tcClone := tc.Clone()
		if len(cl.cc.TLS.ServerName) > 0 {
			tcClone.ServerName = cl.cc.TLS.ServerName
		} else if h, _, err := net.SplitHostPort(host); err == nil {
			tcClone.ServerName = h
		}
		return tls.DialWithDialer(dialer, "tcp", host, tcClone)
	}))

	return nil
}

// Build and set the max versions option
func (cl *Client) buildMaxVersionsOpt() error {
	if len(cl.cc.AsVersion) > 0 {
		var versions *kversion.Versions
		switch cl.cc.AsVersion {
		case "0.8.0":
			versions = kversion.V0_8_0()
		case "0.8.1":
			versions = kversion.V0_8_1()
		case "0.8.2":
			versions = kversion.V0_8_2()
		case "0.9.0":
			versions = kversion.V0_9_0()
		case "0.10.0":
			versions = kversion.V0_10_0()
		case "0.10.1":
			versions = kversion.V0_10_1()
		case "0.10.2":
			versions = kversion.V0_10_2()
		case "0.11.0":
			versions = kversion.V0_11_0()
		case "1.0.0", "1.0":
			versions = kversion.V1_0_0()
		case "1.1.0", "1.1":
			versions = kversion.V1_1_0()
		case "2.0.0", "2.0":
			versions = kversion.V2_0_0()
		case "2.1.0", "2.1":
			versions = kversion.V2_1_0()
		case "2.2.0", "2.2":
			versions = kversion.V2_2_0()
		case "2.3.0", "2.3":
			versions = kversion.V2_3_0()
		case "2.4.0", "2.4":
			versions = kversion.V2_4_0()
		case "2.5.0", "2.5":
			versions = kversion.V2_5_0()
		case "2.6.0", "2.6":
			versions = kversion.V2_6_0()
		case "2.7.0", "2.7":
			versions = kversion.V2_7_0()
		case "2.8.0", "2.8":
			versions = kversion.V2_8_0()
		default:
			return fmt.Errorf("unknown Kafka version %s", cl.cc.AsVersion)
		}
		cl.addOpt(kgo.MaxVersions(versions))
	}

	return nil
}

// Build and set the client log level option
func (cl *Client) buildLogLevelOpt() error {
	var level kgo.LogLevel
	switch ll := strings.ToLower(cl.cc.LogLevel); ll {
	case "none":
		return nil
	case "error":
		level = kgo.LogLevelError
	case "warn":
		level = kgo.LogLevelWarn
	case "info":
		level = kgo.LogLevelInfo
	case "debug":
		level = kgo.LogLevelDebug
	default:
		return fmt.Errorf("invalid log level %q", ll)
	}
	cl.kgoOpts = append(cl.kgoOpts, kgo.WithLogger(kgo.BasicLogger(os.Stderr, level, nil)))

	return nil
}
