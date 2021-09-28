package service

import (
	"context"
	"fmt"
	"time"

	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/meta"
	"github.com/peter-evans/kdef/util/str"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
	"github.com/twmb/franz-go/pkg/kversion"
)

// Cluster metadata
type Metadata struct {
	ClusterId string
	Brokers   meta.Brokers
	Topics    []TopicMetadata
}

// Topic metadata
type TopicMetadata struct {
	Topic                    string
	PartitionAssignments     def.PartitionAssignments
	PartitionRackAssignments def.PartitionRackAssignments
	Exists                   bool
}

// Execute a request for metadata (Kafka 0.8.0+)
func DescribeMetadata(cl *client.Client, topics []string, errorOnNonExistence bool) (*Metadata, error) {
	req := kmsg.NewMetadataRequest()

	// If topics is nil all topics are included
	if topics != nil {
		// If topics is empty no topics are included
		req.Topics = []kmsg.MetadataRequestTopic{}

		for _, topic := range topics {
			t := kmsg.NewMetadataRequestTopic()
			t.Topic = kmsg.StringPtr(topic)
			req.Topics = append(req.Topics, t)
		}
	}

	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return nil, err
	}
	resp := kresp.(*kmsg.MetadataResponse)

	if len(topics) != 0 && len(resp.Topics) != len(topics) {
		return nil, fmt.Errorf("requested %d topics but received %d", len(topics), len(resp.Topics))
	}

	var brokers meta.Brokers
	for _, broker := range resp.Brokers {
		brokers = append(brokers, meta.Broker{
			Id:   broker.NodeID,
			Rack: str.Deref(broker.Rack),
		})
	}

	var tms []TopicMetadata
	for _, t := range resp.Topics {
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
			for _, p := range t.Partitions {
				tm.PartitionAssignments[p.Partition] = p.Replicas
			}

			racksByBroker := brokers.RacksByBroker()
			tm.PartitionRackAssignments = make(def.PartitionRackAssignments, len(t.Partitions))
			for i, p := range tm.PartitionAssignments {
				tm.PartitionRackAssignments[i] = make([]string, len(p))
				for j, r := range p {
					tm.PartitionRackAssignments[i][j] = racksByBroker[r]
				}
			}
		}

		tms = append(tms, tm)
	}

	metadata := Metadata{
		ClusterId: str.Deref(resp.ClusterID),
		Brokers:   brokers,
		Topics:    tms,
	}

	return &metadata, nil
}

// Execute a request to determine if incremental alter configs is supported by the cluster (Kafka 0.10.0+)
func IncrementalAlterConfigsIsSupported(cl *client.Client) (bool, error) {
	r := kmsg.NewIncrementalAlterConfigsRequest()
	return requestIsSupported(cl, r.Key())
}

// Execute a request to determine if a request key is supported by the cluster (Kafka 0.10.0+)
func requestIsSupported(cl *client.Client, requestKey int16) (bool, error) {
	req := kmsg.NewApiVersionsRequest()
	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return false, err
	}
	resp := kresp.(*kmsg.ApiVersionsResponse)
	return kversion.FromApiVersionsResponse(resp).HasKey(requestKey), nil
}

// Execute a request to describe the cluster (Kafka 2.8.0+)
func DescribeCluster(cl *client.Client) (*kmsg.DescribeClusterResponse, error) {
	kresp, err := cl.Client().Request(context.Background(), kmsg.NewPtrDescribeClusterRequest())
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

// Execute describe cluster requests until a minimum number of brokers are alive (Kafka 2.8.0+)
func IsKafkaReady(cl *client.Client, minBrokers int, timeoutSec int) bool {
	timeout := time.After(time.Duration(timeoutSec) * time.Second)

	for {
		select {
		case <-timeout:
			return false
		default:
			resp, err := DescribeCluster(cl)
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
