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

// *** Offset ports in use by tests to allow parallel execution ***

var BrokerApplierTest = ComposeTest{
	ComposeFilePaths: []string{"../../../test/fixtures/compose/1-broker-docker-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10000,
	BrokerPort:       brokerPort + 10000,
	Brokers:          1,
}

var BrokerExporterTest = ComposeTest{
	ComposeFilePaths: []string{"../../../test/fixtures/compose/2-broker-docker-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10100,
	BrokerPort:       brokerPort + 10100,
	Brokers:          2,
}

var BrokersApplierTest = ComposeTest{
	ComposeFilePaths: []string{"../../../test/fixtures/compose/1-broker-docker-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10200,
	BrokerPort:       brokerPort + 10200,
	Brokers:          1,
}

var BrokersExporterTest = ComposeTest{
	ComposeFilePaths: []string{"../../../test/fixtures/compose/1-broker-docker-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10300,
	BrokerPort:       brokerPort + 10300,
	Brokers:          1,
}

var TopicsApplierTest = ComposeTest{
	ComposeFilePaths: []string{"../../../test/fixtures/compose/6-broker-docker-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10400,
	BrokerPort:       brokerPort + 10400,
	Brokers:          6,
}

var TopicsExporterTest = ComposeTest{
	ComposeFilePaths: []string{"../../../test/fixtures/compose/1-broker-docker-compose.yml"},
	ZookeeperPort:    zookeeperPort + 10500,
	BrokerPort:       brokerPort + 10500,
	Brokers:          1,
}
