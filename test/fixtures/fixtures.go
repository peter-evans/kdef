package fixtures

import "fmt"

const (
	brokerPort    = 9092
	zookeeperPort = 2181
)

type ComposeTest struct {
	ComposeFilePaths []string
	ZookeeperPort    int
	BrokerPort       int
	Brokers          int
}

func (t ComposeTest) Env() map[string]string {
	env := map[string]string{
		"ZOOKEEPER_PORT": fmt.Sprintf("%d", t.ZookeeperPort),
	}

	for i := 0; i < t.Brokers; i++ {
		port := t.BrokerPort + i
		env[fmt.Sprintf("BROKER%d_PORT", i+1)] = fmt.Sprintf("%d", port)
	}

	return env
}

// Offset ports in use by tests to allow parallel execution

var TopicsApplierTest = ComposeTest{
	ComposeFilePaths: []string{"../../test/fixtures/compose/multi-broker-docker-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10000,
	BrokerPort:       brokerPort + 10000,
	Brokers:          6,
}

var TopicsExporterTest = ComposeTest{
	ComposeFilePaths: []string{"../../test/fixtures/compose/single-broker-docker-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10100,
	BrokerPort:       brokerPort + 10100,
	Brokers:          1,
}
