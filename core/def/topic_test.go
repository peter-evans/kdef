package def

import (
	"testing"

	"github.com/peter-evans/kdef/test/tutil"
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
