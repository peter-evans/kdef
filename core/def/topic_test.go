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
			wantErr:  "metadata.name must be supplied",
		},
		{
			name: "Tests invalid spec partitions",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
			},
			wantErr: "spec.partitions must be greater than 0",
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
			wantErr: "spec.replicationFactor must be greater than 0",
		},
		{
			name: "Tests a valid TopicDefinition",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 3,
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
