// Package brokers implements operators for brokers definition operations.
package brokers

import (
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
	srv *kafka.Service
}

// Execute executes the export operation.
func (e *exporter) Execute() (res.ExportResults, error) {
	log.Infof("Fetching remote cluster-wide broker configuration...")
	brokersDef, err := e.getBrokersDefinition()
	if err != nil {
		return nil, err
	}

	results := make(res.ExportResults, 1)
	results[0] = res.ExportResult{
		ID:  brokersDef.Metadata.Name,
		Def: brokersDef,
	}

	return results, nil
}

func (e *exporter) getBrokersDefinition() (*def.BrokersDefinition, error) {
	brokerConfigs, err := e.srv.DescribeAllBrokerConfigs()
	if err != nil {
		return nil, err
	}

	brokersDef := def.NewBrokersDefinition(brokerConfigs.ToExportableMap())

	return &brokersDef, nil
}
