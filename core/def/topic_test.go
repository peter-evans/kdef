package def

import (
	"reflect"
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
			name: "Tests specifying assignments and rack assignments together",
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
					RackAssignments: [][]string{
						{"zone-a", "zone-b"},
						{"zone-b", "zone-a"},
					},
				},
			},
			wantErr: "assignments and rack assignments cannot be specified together",
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
			name: "Tests invalid number of rack assignments",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					RackAssignments: [][]string{
						{"zone-a", "zone-b"},
						{"zone-b", "zone-a"},
					},
				},
			},
			wantErr: "number of rack assignments must match partitions",
		},
		{
			name: "Tests invalid number of replicas in rack assignment",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					RackAssignments: [][]string{
						{"zone-a", "zone-b"},
						{"zone-b", "zone-c"},
						{"zone-c"},
					},
				},
			},
			wantErr: "number of replicas in each rack assignment must match replication factor",
		},
		{
			name: "Tests invalid rack id",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        2,
					ReplicationFactor: 2,
					RackAssignments: [][]string{
						{"zone-a", "zone-b"},
						{"zone-b", ""},
					},
				},
			},
			wantErr: "rack ids cannot be an empty string",
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
	brokers := Brokers{
		Broker{Id: 1, Rack: "zone-a"},
		Broker{Id: 2, Rack: "zone-a"},
		Broker{Id: 3, Rack: "zone-b"},
		Broker{Id: 4, Rack: "zone-b"},
		Broker{Id: 5, Rack: "zone-c"},
		Broker{Id: 6, Rack: "zone-c"},
	}

	type args struct {
		brokers Brokers
	}
	tests := []struct {
		name     string
		topicDef TopicDefinition
		args     args
		wantErr  string
	}{
		{
			name: "Tests invalid replication factor",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        20,
					ReplicationFactor: 7,
				},
			},
			args: args{
				brokers: brokers,
			},
			wantErr: "replication factor cannot exceed the number of available brokers",
		},
		{
			name: "Tests an invalid broker id",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        6,
					ReplicationFactor: 2,
					Assignments: [][]int32{
						{1, 2},
						{2, 3},
						{3, 4},
						{4, 5},
						{5, 9},
						{6, 1},
					},
				},
			},
			args: args{
				brokers: brokers,
			},
			wantErr: "invalid broker id \"9\" in assignments",
		},
		{
			name: "Tests an invalid rack id",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        6,
					ReplicationFactor: 2,
					RackAssignments: [][]string{
						{"zone-a", "zone-b"},
						{"zone-b", "zone-c"},
						{"zone-c", "zone-a"},
						{"zone-a", "zone-b"},
						{"zone-b", "zone-z"},
						{"zone-c", "zone-a"},
					},
				},
			},
			args: args{
				brokers: brokers,
			},
			wantErr: "invalid rack id \"zone-z\" in rack assignments",
		},
		{
			name: "Tests when a rack ID is specified more times than available brokers",
			topicDef: TopicDefinition{
				Metadata: TopicMetadataDefinition{
					Name: "foo",
				},
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 3,
					RackAssignments: [][]string{
						{"zone-a", "zone-b", "zone-a"},
						{"zone-b", "zone-c", "zone-b"},
						{"zone-c", "zone-c", "zone-c"},
					},
				},
			},
			args: args{
				brokers: brokers,
			},
			wantErr: "rack id \"zone-c\" contains 2 brokers, but is specified for 3 replicas in partition 2",
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
			args: args{
				brokers: brokers,
			},
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.topicDef.ValidateWithMetadata(tt.args.brokers); !tutil.ErrorContains(err, tt.wantErr) {
				t.Errorf("TopicDefinition.ValidateWithMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_rackAssignmentsDefinitionFromMetadata(t *testing.T) {
	type args struct {
		assignments PartitionAssignments
		brokers     Brokers
	}
	tests := []struct {
		name string
		args args
		want PartitionRackAssignments
	}{
		{
			name: "Tests creation of a rack assignments definition from metadata",
			args: args{
				assignments: [][]int32{
					{1, 5, 3},
					{2, 6, 4},
					{3, 1, 5},
					{4, 2, 6},
				},
				brokers: Brokers{
					Broker{Id: 1, Rack: "zone-a"},
					Broker{Id: 2, Rack: "zone-a"},
					Broker{Id: 3, Rack: "zone-b"},
					Broker{Id: 4, Rack: "zone-b"},
					Broker{Id: 5, Rack: ""}, // Rack id not set on broker
					Broker{Id: 6, Rack: "zone-c"},
				},
			},
			want: [][]string{
				{"zone-a", "", "zone-b"},
				{"zone-a", "zone-c", "zone-b"},
				{"zone-b", "zone-a", ""},
				{"zone-b", "zone-a", "zone-c"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rackAssignmentsDefinitionFromMetadata(tt.args.assignments, tt.args.brokers); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rackAssignmentsDefinitionFromMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}
