// Package harness implements test harnesses for integration tests.
package harness

import "fmt"

const (
	brokerPort    = 9092
	zookeeperPort = 2181
)

// ComposeHarness represents a Docker compose based test harness.
type ComposeHarness struct {
	ComposeFilePaths []string
	ZookeeperPort    int
	BrokerPort       int
	Brokers          int
}

// Env returns an environment variable map.
func (t ComposeHarness) Env() map[string]string {
	env := map[string]string{
		"ZOOKEEPER_PORT": fmt.Sprintf("%d", t.ZookeeperPort),
	}

	for i := 0; i < t.Brokers; i++ {
		port := t.BrokerPort + i
		env[fmt.Sprintf("BROKER%d_PORT", i+1)] = fmt.Sprintf("%d", port)
	}

	return env
}

// *** Offset ports in use by tests to allow parallel execution ***

// BrokerApplier represents the compose harness for the broker applier tests.
var BrokerApplier = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/1-broker-plaintext-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10000,
	BrokerPort:       brokerPort + 10000,
	Brokers:          1,
}

// BrokerExporter represents the compose harness for the broker exporter tests.
var BrokerExporter = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/2-broker-plaintext-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10100,
	BrokerPort:       brokerPort + 10100,
	Brokers:          2,
}

// BrokersApplier represents the compose harness for the brokers applier tests.
var BrokersApplier = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/1-broker-plaintext-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10200,
	BrokerPort:       brokerPort + 10200,
	Brokers:          1,
}

// BrokersExporter represents the compose harness for the brokers exporter tests.
var BrokersExporter = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/1-broker-plaintext-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10300,
	BrokerPort:       brokerPort + 10300,
	Brokers:          1,
}

// TopicApplier represents the compose harness for the topic applier tests.
var TopicApplier = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/6-broker-plaintext-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10400,
	BrokerPort:       brokerPort + 10400,
	Brokers:          6,
}

// TopicExporter represents the compose harness for the topic exporter tests.
var TopicExporter = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/1-broker-plaintext-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10500,
	BrokerPort:       brokerPort + 10500,
	Brokers:          1,
}

// ACLApplier represents the compose harness for the acl applier tests.
var ACLApplier = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/1-broker-sasl-plain-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10200,
	BrokerPort:       brokerPort + 10200,
	Brokers:          1,
}

// ACLExporter represents the compose harness for the acl exporter tests.
var ACLExporter = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/1-broker-sasl-plain-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10300,
	BrokerPort:       brokerPort + 10300,
	Brokers:          1,
}
