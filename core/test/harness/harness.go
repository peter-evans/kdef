package harness

import "fmt"

const (
	brokerPort    = 9092
	zookeeperPort = 2181
)

type ComposeHarness struct {
	ComposeFilePaths []string
	ZookeeperPort    int
	BrokerPort       int
	Brokers          int
}

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

var BrokerApplier = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/1-broker-plaintext-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10000,
	BrokerPort:       brokerPort + 10000,
	Brokers:          1,
}

var BrokerExporter = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/2-broker-plaintext-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10100,
	BrokerPort:       brokerPort + 10100,
	Brokers:          2,
}

var BrokersApplier = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/1-broker-plaintext-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10200,
	BrokerPort:       brokerPort + 10200,
	Brokers:          1,
}

var BrokersExporter = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/1-broker-plaintext-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10300,
	BrokerPort:       brokerPort + 10300,
	Brokers:          1,
}

var TopicsApplier = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/6-broker-plaintext-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10400,
	BrokerPort:       brokerPort + 10400,
	Brokers:          6,
}

var TopicsExporter = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/1-broker-plaintext-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10500,
	BrokerPort:       brokerPort + 10500,
	Brokers:          1,
}

var ACLApplier = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/1-broker-sasl-plain-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10200,
	BrokerPort:       brokerPort + 10200,
	Brokers:          1,
}

var ACLExporter = ComposeHarness{
	ComposeFilePaths: []string{"../../test/fixtures/compose/1-broker-sasl-plain-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10300,
	BrokerPort:       brokerPort + 10300,
	Brokers:          1,
}
