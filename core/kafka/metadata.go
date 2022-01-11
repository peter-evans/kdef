// Package kafka implements the Kafka service handling requests and responses.
package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/meta"
	"github.com/peter-evans/kdef/core/util/str"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
	"github.com/twmb/franz-go/pkg/kversion"
)

// Metadata represents cluster metadata.
type Metadata struct {
	ClusterID string
	Brokers   meta.Brokers
	Topics    []TopicMetadata
}

// TopicMetadata represents topic metadata.
type TopicMetadata struct {
	Topic                string
	PartitionAssignments def.PartitionAssignments
	PartitionRacks       def.PartitionRacks
	PartitionLeaders     def.PartitionLeaders
	PartitionISR         def.PartitionAssignments
	Exists               bool
}

// describeMetadata executes a request for metadata (Kafka 0.8.0+).
func describeMetadata(
	ctx context.Context,
	cl *client.Client,
	topics []string,
	errorOnNonExistence bool,
) (*Metadata, error) {
	req := kmsg.NewMetadataRequest()

	// If topics is nil all topics are included.
	if topics != nil {
		// If topics is empty no topics are included.
		req.Topics = []kmsg.MetadataRequestTopic{}

		for _, topic := range topics {
			t := kmsg.NewMetadataRequestTopic()
			t.Topic = kmsg.StringPtr(topic)
			req.Topics = append(req.Topics, t)
		}
	}

	kresp, err := cl.Client.Request(ctx, &req)
	if err != nil {
		return nil, err
	}
	resp := kresp.(*kmsg.MetadataResponse)

	if len(topics) != 0 && len(resp.Topics) != len(topics) {
		return nil, fmt.Errorf("requested %d topic(s) but received %d", len(topics), len(resp.Topics))
	}

	var brokers meta.Brokers
	for _, broker := range resp.Brokers {
		brokers = append(brokers, meta.Broker{
			ID:   broker.NodeID,
			Rack: str.Deref(broker.Rack),
		})
	}

	tms := make([]TopicMetadata, len(resp.Topics))
	for i, t := range resp.Topics {
		exists := t.ErrorCode != kerr.UnknownTopicOrPartition.Code
		if err := kerr.ErrorForCode(t.ErrorCode); err != nil && (errorOnNonExistence || exists) {
			return nil, err
		}

		tm := TopicMetadata{
			Topic:  *t.Topic,
			Exists: exists,
		}

		if exists {
			tm.PartitionAssignments = make(def.PartitionAssignments, len(t.Partitions))
			tm.PartitionLeaders = make([]int32, len(t.Partitions))
			tm.PartitionISR = make(def.PartitionAssignments, len(t.Partitions))
			for _, p := range t.Partitions {
				tm.PartitionAssignments[p.Partition] = p.Replicas
				tm.PartitionLeaders[p.Partition] = p.Leader
				tm.PartitionISR[p.Partition] = p.ISR
			}

			racksByBroker := brokers.RacksByBroker()
			tm.PartitionRacks = make(def.PartitionRacks, len(t.Partitions))
			for i, p := range tm.PartitionAssignments {
				tm.PartitionRacks[i] = make([]string, len(p))
				for j, r := range p {
					tm.PartitionRacks[i][j] = racksByBroker[r]
				}
			}
		}

		tms[i] = tm
	}

	metadata := Metadata{
		ClusterID: str.Deref(resp.ClusterID),
		Brokers:   brokers,
		Topics:    tms,
	}

	return &metadata, nil
}

// requestIsSupported executes a request to determine if a request key is supported by the cluster (Kafka 0.10.0+).
func requestIsSupported(ctx context.Context, cl *client.Client, requestKey int16) (bool, error) {
	req := kmsg.NewApiVersionsRequest()
	kresp, err := cl.Client.Request(ctx, &req)
	if err != nil {
		return false, err
	}
	resp := kresp.(*kmsg.ApiVersionsResponse)
	return kversion.FromApiVersionsResponse(resp).HasKey(requestKey), nil
}

// describeCluster executes a request to describe the cluster (Kafka 2.8.0+).
func describeCluster(ctx context.Context, cl *client.Client) (*kmsg.DescribeClusterResponse, error) {
	kresp, err := cl.Client.Request(ctx, kmsg.NewPtrDescribeClusterRequest())
	if err != nil {
		return nil, err
	}
	resp := kresp.(*kmsg.DescribeClusterResponse)

	if err := kerr.ErrorForCode(resp.ErrorCode); err != nil {
		errMsg := err.Error()
		if resp.ErrorMessage != nil {
			errMsg = fmt.Sprintf("%s: %s", errMsg, *resp.ErrorMessage)
		}
		return nil, fmt.Errorf(errMsg)
	}

	return resp, nil
}

// isKafkaReady executes describe cluster requests until a minimum number of brokers are alive (Kafka 2.8.0+).
func isKafkaReady(ctx context.Context, cl *client.Client, minBrokers int, timeoutSec int) bool {
	timeout := time.After(time.Duration(timeoutSec) * time.Second)

	for {
		select {
		case <-timeout:
			return false
		default:
			resp, err := describeCluster(ctx, cl)
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
