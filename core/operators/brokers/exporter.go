package brokers

import (
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/res"
	"github.com/peter-evans/kdef/core/service"
)

// An exporter handling the export operation
type exporter struct {
	// constructor params
	cl *client.Client
}

// Create a new exporter
func NewExporter(
	cl *client.Client,
) *exporter {
	return &exporter{
		cl: cl,
	}
}

// Execute the export operation
func (e *exporter) Execute() (res.ExportResults, error) {
	log.Info("Fetching broker configuration...")
	brokersDef, err := e.getBrokersDefinition()
	if err != nil {
		return nil, err
	}

	results := make(res.ExportResults, 1)
	results[0] = res.ExportResult{
		Id:  brokersDef.Metadata.Name,
		Def: brokersDef,
	}

	return results, nil
}

// Return the brokers definition
func (e *exporter) getBrokersDefinition() (*def.BrokersDefinition, error) {
	metadata, err := service.DescribeMetadata(e.cl, []string{}, true)
	if err != nil {
		return nil, err
	}

	name := metadata.ClusterId
	if len(name) == 0 {
		name = "brokers"
	}

	brokerConfigs, err := service.DescribeAllBrokerConfigs(e.cl)
	if err != nil {
		return nil, err
	}

	brokersDef := def.NewBrokersDefinition(name, brokerConfigs.ToMap())

	return &brokersDef, nil
}
