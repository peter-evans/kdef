package def

import (
	"testing"

	"github.com/peter-evans/kdef/test/tutil"
	"github.com/twmb/franz-go/pkg/kmsg"
)

func TestTopicDefinition_Validate(t *testing.T) {
	tests := []struct {
		name     string
		topicDef TopicDefinition
		wantErr  string
	}{
		{
			name:     "Tests missing metadata name",
			topicDef: TopicDefinition{},
			wantErr:  "metadata name must be supplied",
		},
		{
			name: "Tests invalid spec partitions",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
			},
			wantErr: "partitions must be greater than 0",
		},
		{
			name: "Tests invalid spec replication factor",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions: 3,
				},
			},
			wantErr: "replication factor must be greater than 0",
		},
		{
			name: "Tests invalid number of assignments",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					Assignments: [][]int32{
						{1, 2},
						{2, 3},
					},
				},
			},
			wantErr: "number of replica assignments must match partitions",
		},
		{
			name: "Tests invalid number of replicas in assignment",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					Assignments: [][]int32{
						{1, 2},
						{2, 3},
						{3},
					},
				},
			},
			wantErr: "number of replicas in each assignment must match replication factor",
		},
		{
			name: "Tests duplicate brokers in replica assignment",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					Assignments: [][]int32{
						{1, 2},
						{2, 3},
						{3, 3},
					},
				},
			},
			wantErr: "a replica assignment cannot contain duplicate brokers",
		},
		{
			name: "Tests invalid reassignment await timeout",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					Reassignment: TopicReassignmentDefinition{
						AwaitTimeoutSec: -10,
					},
				},
			},
			wantErr: "reassignment await timeout seconds must be greater or equal to 0",
		},
		{
			name: "Tests a valid TopicDefinition",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					Assignments: [][]int32{
						{1, 2},
						{2, 3},
						{3, 1},
					},
					Reassignment: TopicReassignmentDefinition{
						AwaitTimeoutSec: 30,
					},
				},
			},
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.topicDef.Validate(); !tutil.ErrorContains(err, tt.wantErr) {
				t.Errorf("TopicDefinition.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTopicDefinition_ValidateWithMetadata(t *testing.T) {
	metadataResp := &kmsg.MetadataResponse{
		Brokers: []kmsg.MetadataResponseBroker{
			{
				NodeID: 1,
			},
			{
				NodeID: 2,
			},
			{
				NodeID: 3,
			},
		},
	}

	type args struct {
		metadata *kmsg.MetadataResponse
	}
	tests := []struct {
		name     string
		topicDef TopicDefinition
		args     args
		wantErr  string
	}{
		{
			name: "Tests an invalid broker id",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					Assignments: [][]int32{
						{1, 2},
						{2, 4},
						{3, 1},
					},
				},
			},
			args: args{
				metadata: metadataResp,
			},
			wantErr: "invalid broker id \"4\" in assignments",
		},
		{
			name: "Tests valid broker ids",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					Assignments: [][]int32{
						{1, 2},
						{2, 3},
						{3, 1},
					},
				},
			},
			args: args{
				metadata: metadataResp,
			},
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.topicDef.ValidateWithMetadata(tt.args.metadata); !tutil.ErrorContains(err, tt.wantErr) {
				t.Errorf("TopicDefinition.ValidateWithMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
