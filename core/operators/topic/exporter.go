package topic

import (
	"regexp"
	"strings"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/kafka"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/core/model/res"
)

// Options to configure an exporter
type ExporterOptions struct {
	Match           string
	Exclude         string
	IncludeInternal bool
	Assignments     opt.Assignments
}

// Create a new exporter
func NewExporter(
	cl *client.Client,
	opts ExporterOptions,
) *exporter {
	return &exporter{
		srv:  kafka.NewService(cl),
		opts: opts,
	}
}

// An exporter handling the export operation
type exporter struct {
	// constructor params
	srv  *kafka.Service
	opts ExporterOptions
}

// Execute the export operation
func (e *exporter) Execute() (res.ExportResults, error) {
	log.Info("Fetching topics...")
	topicDefs, err := e.getTopicDefinitions()
	if err != nil {
		return nil, err
	}

	if len(topicDefs) == 0 {
		return nil, nil
	}

	results := make(res.ExportResults, len(topicDefs))
	for i, topicDef := range topicDefs {
		results[i] = res.ExportResult{
			Id:  topicDef.Metadata.Name,
			Def: topicDef,
		}
	}

	results.Sort()

	return results, nil
}

// Return topic definitions for existing topics in a cluster
func (e *exporter) getTopicDefinitions() ([]def.TopicDefinition, error) {
	metadata, err := e.srv.DescribeMetadata(nil, true)
	if err != nil {
		return nil, err
	}

	topicNames := []string{}
	topicMetadataMap := map[string]kafka.TopicMetadata{}
	for _, t := range metadata.Topics {
		topicNames = append(topicNames, t.Topic)
		topicMetadataMap[t.Topic] = t
	}

	resourceConfigs, err := e.srv.DescribeTopicConfigs(topicNames)
	if err != nil {
		return nil, err
	}

	topicConfigsMapMap := map[string]def.ConfigsMap{}
	for _, resource := range resourceConfigs {
		topicConfigsMapMap[resource.ResourceName] = resource.Configs.ToExportableMap()
	}

	matchRegExp, err := regexp.Compile(e.opts.Match)
	if err != nil {
		return nil, err
	}
	excludeRegExp, err := regexp.Compile(e.opts.Exclude)
	if err != nil {
		return nil, err
	}

	topicDefs := []def.TopicDefinition{}
	for _, topic := range topicNames {
		// Kafka internal topics are prefixed by double underscores
		// Confluent Schema Registry uses a single underscore
		if strings.HasPrefix(topic, "_") && !e.opts.IncludeInternal {
			continue
		}
		if !matchRegExp.MatchString(topic) {
			continue
		}
		if excludeRegExp.MatchString(topic) {
			continue
		}
		topicDefs = append(
			topicDefs,
			def.NewTopicDefinition(
				topic,
				topicMetadataMap[topic].PartitionAssignments,
				topicMetadataMap[topic].PartitionRackAssignments,
				topicConfigsMapMap[topic],
				metadata.Brokers,
				e.opts.Assignments == opt.BrokerAssignments,
				e.opts.Assignments == opt.RackAssignments,
			),
		)
	}

	return topicDefs, nil
}
