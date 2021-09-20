package req

import (
	"context"
	"fmt"
	"time"

	"github.com/peter-evans/kdef/client"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
	"github.com/twmb/franz-go/pkg/kversion"
)

// Execute a request for metadata (Kafka 0.8.0+)
func RequestMetadata(cl *client.Client, topics []string, validateResponse bool) (*kmsg.MetadataResponse, error) {
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

	return resp, nil
}

// Execute a request to determine if a request key is supported by the cluster (Kafka 0.10.0+)
func RequestSupported(cl *client.Client, requestKey int16) (bool, error) {
	req := kmsg.NewApiVersionsRequest()
	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return false, err
	}
	resp := kresp.(*kmsg.ApiVersionsResponse)
	return kversion.FromApiVersionsResponse(resp).HasKey(requestKey), nil
}

// Execute a request to describe the cluster (Kafka 2.8.0+)
func RequestDescribeCluster(cl *client.Client) (*kmsg.DescribeClusterResponse, error) {
	kresp, err := cl.Client().Request(context.Background(), kmsg.NewPtrDescribeClusterRequest())
	if err != nil {
		return nil, err
	}
	resp := kresp.(*kmsg.DescribeClusterResponse)

	if err := kerr.ErrorForCode(resp.ErrorCode); err != nil {
		return nil, fmt.Errorf(*resp.ErrorMessage)
	}

	return resp, nil
}

// Execute describe cluster requests until a minimum number of brokers are alive (Kafka 2.8.0+)
func IsKafkaReady(cl *client.Client, minBrokers int, timeoutSec int) bool {
	timeout := time.After(time.Duration(timeoutSec) * time.Second)

	for {
		select {
		case <-timeout:
			return false
		default:
			resp, err := RequestDescribeCluster(cl)
			if err == nil {
				if len(resp.Brokers) >= minBrokers {
					return true
				}
			}
			time.Sleep(2 * time.Second)
			continue
		}
	}
}
