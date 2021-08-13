package topics

import (
	"context"
	"fmt"

	"github.com/peter-evans/kdef/client"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
	"github.com/twmb/franz-go/pkg/kversion"
)

// Execute a request for topic metadata (Kafka 0.8.0+)
func requestTopicMetadata(cl *client.Client, topics []string, validateResponse bool) ([]kmsg.MetadataResponseTopic, error) {
	req := kmsg.NewMetadataRequest()

	for _, topic := range topics {
		t := kmsg.NewMetadataRequestTopic()
		t.Topic = kmsg.StringPtr(topic)
		req.Topics = append(req.Topics, t)
	}

	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return nil, err
	}
	resp := kresp.(*kmsg.MetadataResponse)

	if len(topics) != 0 && len(resp.Topics) != len(topics) {
		return nil, fmt.Errorf("requested %d topics but received %d", len(topics), len(resp.Topics))
	}

	if validateResponse {
		for _, topic := range resp.Topics {
			if err := kerr.ErrorForCode(topic.ErrorCode); err != nil {
				return nil, err
			}
		}
	}

	return resp.Topics, nil
}

// Execute a request to describe topic configs (Kafka 0.11.0+)
func requestDescribeTopicConfigs(cl *client.Client, topics []string) ([]kmsg.DescribeConfigsResponseResource, error) {
	req := kmsg.NewDescribeConfigsRequest()

	for _, topic := range topics {
		res := kmsg.NewDescribeConfigsRequestResource()
		res.ResourceType = kmsg.ConfigResourceTypeTopic
		res.ResourceName = topic
		req.Resources = append(req.Resources, res)
	}

	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return nil, err
	}
	resp := kresp.(*kmsg.DescribeConfigsResponse)

	if len(resp.Resources) != len(topics) {
		return nil, fmt.Errorf("requested %d resources but received %d", len(topics), len(resp.Resources))
	}

	for _, resource := range resp.Resources {
		if err := kerr.ErrorForCode(resource.ErrorCode); err != nil {
			return nil, fmt.Errorf(*resource.ErrorMessage)
		}
	}

	return resp.Resources, nil
}

// Execute a request to determine if a request key is supported by the cluster (Kafka 0.10.0+)
func requestSupported(cl *client.Client, requestKey int16) (bool, error) {
	req := kmsg.NewApiVersionsRequest()
	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return false, err
	}
	resp := kresp.(*kmsg.ApiVersionsResponse)
	return kversion.FromApiVersionsResponse(resp).HasKey(requestKey), nil
}
