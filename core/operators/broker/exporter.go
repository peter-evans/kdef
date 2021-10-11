package broker

import (
	"fmt"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/kafka"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/res"
)

// Create a new exporter
func NewExporter(
	cl *client.Client,
) *exporter {
	return &exporter{
		srv: kafka.NewService(cl),
	}
}

// An exporter handling the export operation
type exporter struct {
	// constructor params
	srv *kafka.Service
}

// Execute the export operation
func (e *exporter) Execute() (res.ExportResults, error) {
	log.Info("Fetching per-broker configuration...")
	brokerDefs, err := e.getBrokerDefinitions()
	if err != nil {
		return nil, err
	}

	if len(brokerDefs) == 0 {
		// Should not happen. There should always be at least one broker.
		return nil, nil
	}

	results := make(res.ExportResults, len(brokerDefs))
	for i, brokerDef := range brokerDefs {
		results[i] = res.ExportResult{
			Id:  brokerDef.Metadata.Name,
			Def: brokerDef,
		}
	}

	results.Sort()

	return results, nil
}

// Return the broker definitions
func (e *exporter) getBrokerDefinitions() ([]def.BrokerDefinition, error) {
	metadata, err := e.srv.DescribeMetadata([]string{}, true)
	if err != nil {
		return nil, err
	}

	brokerDefs := []def.BrokerDefinition{}
	for _, broker := range metadata.Brokers {
		brokerIdStr := fmt.Sprint(broker.Id)
		brokerConfigs, err := e.srv.DescribeBrokerConfigs(brokerIdStr)
		if err != nil {
			return nil, err
		}
		brokerDefs = append(
			brokerDefs, def.NewBrokerDefinition(brokerIdStr, brokerConfigs.ToExportableMap()),
		)
	}

	return brokerDefs, nil
}
