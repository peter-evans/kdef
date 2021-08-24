package topics

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/def"
	"github.com/peter-evans/kdef/core/req"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// Flags to configure an exporter
type ExporterFlags struct {
	OutputDir       string
	Overwrite       bool
	Match           string
	Exclude         string
	IncludeInternal bool
}

// An exporter handling the export operation
type exporter struct {
	// constructor params
	cl    *client.Client
	flags ExporterFlags
}

// Creates a new exporter
func NewExporter(
	cl *client.Client,
	flags ExporterFlags,
) *exporter {
	return &exporter{
		cl:    cl,
		flags: flags,
	}
}

// Executes the export operation
func (e *exporter) Execute() (int, error) {
	log.Info("Fetching topics...")
	topicDefs, err := e.getTopicDefinitions()
	if err != nil {
		return 0, err
	}

	topicCount := len(topicDefs)
	if topicCount == 0 {
		log.Info("No topics found")
		return 0, nil
	}

	for index, topicDef := range topicDefs {
		log.Info("Exporting topic definition %d of %d", index+1, topicCount)

		yaml, err := topicDef.YAML()
		if err != nil {
			return 0, err
		}

		if e.flags.OutputDir != "" {
			outputPath := filepath.Join(e.flags.OutputDir, fmt.Sprintf("%s.yml", topicDef.Metadata.Name))

			if !e.flags.Overwrite {
				if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
					log.Info("Skipping overwrite of existing file %q", outputPath)
					continue
				}
			}

			log.Info("Writing topic definition file %q", outputPath)
			if err = ioutil.WriteFile(outputPath, []byte(yaml), 0644); err != nil {
				return 0, err
			}
		} else {
			// Ignores --quiet
			fmt.Printf("---\n%s", yaml)
		}
	}

	return topicCount, nil
}

// Returns topic definitions from existing topics in a cluster
func (e *exporter) getTopicDefinitions() ([]def.TopicDefinition, error) {
	topicMetadata, err := req.RequestTopicMetadata(e.cl, []string{}, true)
	if err != nil {
		return nil, err
	}

	topicNames := []string{}
	for _, topic := range topicMetadata {
		topicNames = append(topicNames, topic.Topic)
	}

	topicConfigs, err := req.RequestDescribeTopicConfigs(e.cl, topicNames)
	if err != nil {
		return nil, err
	}

	topicConfigsMap := map[string]kmsg.DescribeConfigsResponseResource{}
	for _, resource := range topicConfigs {
		topicConfigsMap[resource.ResourceName] = resource
	}

	matchRegExp, err := regexp.Compile(e.flags.Match)
	if err != nil {
		return nil, err
	}
	excludeRegExp, err := regexp.Compile(e.flags.Exclude)
	if err != nil {
		return nil, err
	}

	topicDefs := []def.TopicDefinition{}
	for _, topic := range topicMetadata {
		// Kafka internal topics are prefixed by double underscores
		// Confluent Schema Registry uses a single underscore
		if strings.HasPrefix(topic.Topic, "_") && !e.flags.IncludeInternal {
			continue
		}
		if !matchRegExp.MatchString(topic.Topic) {
			continue
		}
		if excludeRegExp.MatchString(topic.Topic) {
			continue
		}
		topicDefs = append(
			topicDefs,
			def.TopicDefinitionFromMetadata(topic, topicConfigsMap[topic.Topic]),
		)
	}

	return topicDefs, nil
}
