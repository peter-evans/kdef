package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/peter-evans/kdef/cli/scanner"
	"github.com/peter-evans/kdef/core/client"
)

const intro = `
Welcome to kdef!

This short interactive prompt will guide through creating a configuration
file for your cluster.

---`

const promptSeedBrokers = `
Specify one or more seed broker addresses that kdef will connect to. They can
be entered over multiple lines or comma delimited on one line. An empty line
continues to the next step.

e.g. > b1-mycluster.example.com:9092
`

const promptServerName = `
When connecting via TLS, by default the client will use the hostname or IP
address of the connected broker as the TLS server name.
`

func Configure() error {
	s := scanner.New()

	fmt.Println(intro)

	fmt.Println(promptSeedBrokers)
	seedBrokers := s.PromptMultiline(">", []string{"localhost:9092"})
	rootConfig := map[string]interface{}{
		"seedBrokers": seedBrokers,
	}

	tlsConfig := configureTLS(s)
	saslConfig := configureSASL(s)

	// Load config
	k := koanf.New(".")
	if err := k.Load(confmap.Provider(rootConfig, "."), nil); err != nil {
		return err
	}
	if err := k.Load(confmap.Provider(tlsConfig, "."), nil); err != nil {
		return err
	}
	if err := k.Load(confmap.Provider(saslConfig, "."), nil); err != nil {
		return err
	}

	// Unmarshal to config struct
	cc := &client.Config{}
	if err := k.UnmarshalWithConf("", cc, koanf.UnmarshalConf{Tag: "json"}); err != nil {
		return fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// Marshal to yaml
	y, err := yaml.Marshal(cc)
	if err != nil {
		return fmt.Errorf("failed to marshal config to yaml: %v", err)
	}

	// Prompt for config output path
	fmt.Printf("\nEnter the path where the configuration file should be written.\n\n")
	defaultConfigPath := DefaultConfigPath()
	configPath := s.PromptLine(fmt.Sprintf("(%s) >", defaultConfigPath), defaultConfigPath)

	// Check for existing file
	if _, err := os.Stat(configPath); !errors.Is(err, os.ErrNotExist) {
		fmt.Printf("\nOverwrite the existing file? (Yes/No)\n\n")
		if !s.PromptYesNo("(No) >", false) {
			fmt.Printf("\nPrinting configuration file to stdout:\n\n")
			fmt.Printf("---\n%s", string(y))
			return nil
		}
	}

	// Write the config file
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("failed to create configuration directory: %v", err)
	}
	if err = ioutil.WriteFile(configPath, y, 0o666); err != nil {
		return fmt.Errorf("failed to write configuration file: %v", err)
	}
	fmt.Printf("\nCreated configuration file at %s\n", configPath)

	return nil
}

func configureTLS(s *scanner.Scanner) map[string]interface{} {
	tlsConfig := map[string]interface{}{}

	fmt.Printf("\nDoes connection to your cluster broker(s) require TLS? (Yes/No)\n\n")
	if s.PromptYesNo("(No) >", false) {
		tlsConfig["tls.enabled"] = true

		fmt.Printf("\nDoes connecting via TLS require a custom CA cert? (Yes/No)\n\n")
		if s.PromptYesNo("(No) >", false) {
			fmt.Printf("\nEnter the path to your CA cert.\n\n")
			tlsConfig["tls.caCertPath"] = s.PromptLine(">", "")
		}

		fmt.Printf("\nDoes connecting via TLS require a client cert and key? (Yes/No)\n\n")
		if s.PromptYesNo("(No) >", false) {
			fmt.Printf("\nEnter the path to your client cert.\n\n")
			tlsConfig["tls.clientCertPath"] = s.PromptLine(">", "")
			fmt.Printf("\nEnter the path to your client key.\n\n")
			tlsConfig["tls.clientKeyPath"] = s.PromptLine(">", "")
		}

		fmt.Print(promptServerName)
		fmt.Printf("Does connecting via TLS require a distinct server name? (Yes/No)\n\n")
		if s.PromptYesNo("(No) >", false) {
			fmt.Printf("\nEnter the distinct server name.\n\n")
			tlsConfig["tls.serverName"] = s.PromptLine(">", "")
		}
	}

	return tlsConfig
}

func configureSASL(s *scanner.Scanner) map[string]interface{} {
	saslConfig := map[string]interface{}{}

	fmt.Printf("\nDoes connection to your cluster broker(s) require SASL? (Yes/No)\n\n")
	if s.PromptYesNo("(No) >", false) {
		fmt.Printf("\nEnter the required SASL method. (plain, scram-sha-256, scram-sha-512, aws-msk-iam)\n\n")
		method := s.PromptLine(">", "")

		var isSCRAM, isAWS bool
		switch method {
		case "plain":
			saslConfig["sasl.method"] = method
		case "scram-sha-256", "scram-sha-512":
			saslConfig["sasl.method"] = method
			isSCRAM = true
		case "aws-msk-iam":
			saslConfig["sasl.method"] = method
			isAWS = true
		default:
			fmt.Printf("Unrecognised SASL method %q. Continuing to next step.\n", method)
			return saslConfig
		}

		if !isAWS {
			fmt.Printf("\nEnter the SASL username.\n\n")
			saslConfig["sasl.user"] = s.PromptLine(">", "")
			fmt.Printf("\nEnter the SASL password.\n\n")
			saslConfig["sasl.pass"] = s.PromptLine(">", "")
		}

		if isSCRAM {
			fmt.Printf("\nIs this SASL from a delegation token? (Yes/No)\n\n")
			if s.PromptYesNo("(No) >", false) {
				saslConfig["sasl.isToken"] = true
			}
		}
	}

	return saslConfig
}
