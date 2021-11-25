// Package broker implements operators for broker definition operations.
package broker

import (
	"fmt"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/kafka"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/res"
)

// NewExporter creates a new exporter.
func NewExporter(
	cl *client.Client,
) *exporter { //revive:disable-line:unexported-return
	return &exporter{
		srv: kafka.NewService(cl),
	}
}

type exporter struct {
	// constructor params
	srv *kafka.Service
}

// Execute executes the export operation.
func (e *exporter) Execute() (res.ExportResults, error) {
	log.Infof("Fetching remote per-broker configuration...")
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
			ID:  brokerDef.Metadata.Name,
			Def: brokerDef,
		}
	}

	results.Sort()

	return results, nil
}

func (e *exporter) getBrokerDefinitions() ([]def.BrokerDefinition, error) {
	metadata, err := e.srv.DescribeMetadata([]string{}, true)
	if err != nil {
		return nil, err
	}

	brokerDefs := []def.BrokerDefinition{}
	for _, broker := range metadata.Brokers {
		brokerIDStr := fmt.Sprint(broker.ID)
		brokerConfigs, err := e.srv.DescribeBrokerConfigs(brokerIDStr)
		if err != nil {
			return nil, err
		}
		brokerDefs = append(
			brokerDefs, def.NewBrokerDefinition(
				def.ResourceMetadataDefinition{
					Name: brokerIDStr,
				},
				brokerConfigs.ToExportableMap(),
			),
		)
	}

	return brokerDefs, nil
}
