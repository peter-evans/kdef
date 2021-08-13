package topics

import (
	"errors"

	"github.com/ghodss/yaml"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// Top-level topic definition
type TopicDefinition struct {
	ApiVersion string                  `json:"apiVersion"`
	Kind       string                  `json:"kind"`
	Metadata   TopicMetadataDefinition `json:"metadata"`
	Spec       TopicSpecDefinition     `json:"spec"`
}

// Topic metadata definition
type TopicMetadataDefinition struct {
	Labels TopicMetadataLabels `json:"labels,omitempty"`
	Name   string              `json:"name"`
}

// Topic metadata labels
type TopicMetadataLabels map[string]string

// Topic spec definition
type TopicSpecDefinition struct {
	Configs           TopicConfigDefinition `json:"configs,omitempty"`
	Partitions        int                   `json:"partitions"`
	ReplicationFactor int                   `json:"replicationFactor"`
}

// Topic config definition
type TopicConfigDefinition map[string]*string

// Converts a topic definition to YAML
func (t TopicDefinition) YAML() (string, error) {
	y, err := yaml.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(y), nil
}

// Validate a topic definition
func (t TopicDefinition) Validate() error {
	if len(t.Metadata.Name) == 0 {
		return errors.New("metadata.name must be supplied")
	}

	if t.Spec.Partitions <= 0 {
		return errors.New("spec.partitions must be greater than 0")
	}

	if t.Spec.ReplicationFactor <= 0 {
		return errors.New("spec.replicationFactor must be greater than 0")
	}

	return nil
}

// Create a topic definition from metadata and config
func topicDefinitionFromMetadata(metadata kmsg.MetadataResponseTopic, topicConfig kmsg.DescribeConfigsResponseResource) TopicDefinition {
	topicConfigsDef := TopicConfigDefinition{}
	for _, config := range topicConfig.Configs {
		topicConfigsDef[config.Name] = config.Value
	}

	return TopicDefinition{
		ApiVersion: "v1",
		Kind:       "topic",
		Metadata: TopicMetadataDefinition{
			Name: metadata.Topic,
		},
		Spec: TopicSpecDefinition{
			Partitions:        len(metadata.Partitions),
			ReplicationFactor: len(metadata.Partitions[0].Replicas),
			Configs:           topicConfigsDef,
		},
	}
}
