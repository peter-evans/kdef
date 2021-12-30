// Package def implements definitions for Kafka resources.
package def

import (
	"testing"

	"github.com/peter-evans/kdef/core/model/meta"
	"github.com/peter-evans/kdef/core/test/tutil"
)

func TestTopicDefinition_Validate(t *testing.T) {
	resDef := ResourceDefinition{
		APIVersion: "v1",
		Kind:       "topic",
		Metadata: ResourceMetadataDefinition{
			Name: "foo",
		},
	}

	tests := []struct {
		name     string
		topicDef TopicDefinition
		wantErr  string
	}{
		{
			name: "Tests invalid spec partitions",
			topicDef: TopicDefinition{
				ResourceDefinition: resDef,
			},
			wantErr: "partitions must be greater than 0",
		},
		{
			name: "Tests invalid spec replication factor",
			topicDef: TopicDefinition{
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions: 3,
				},
			},
			wantErr: "replication factor must be greater than 0",
		},
		{
			name: "Tests invalid number of assignments",
			topicDef: TopicDefinition{
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					Assignments: PartitionAssignments{
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
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					Assignments: PartitionAssignments{
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
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					Assignments: PartitionAssignments{
						{1, 2},
						{2, 3},
						{3, 3},
					},
				},
			},
			wantErr: "a replica assignment cannot contain duplicate brokers",
		},
		{
			name: "Tests invalid selection",
			topicDef: TopicDefinition{
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					ManagedAssignments: &ManagedAssignmentsDefinition{
						Selection: "foo",
					},
				},
			},
			wantErr: "selection must be one of",
		},
		{
			name: "Tests invalid number of rack constraints",
			topicDef: TopicDefinition{
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					ManagedAssignments: &ManagedAssignmentsDefinition{
						Selection: "topic-cluster-use",
						RackConstraints: PartitionRacks{
							{"zone-a", "zone-b"},
							{"zone-b", "zone-a"},
						},
					},
				},
			},
			wantErr: "number of rack constraints must match partitions",
		},
		{
			name: "Tests invalid number of replicas in rack constraints",
			topicDef: TopicDefinition{
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					ManagedAssignments: &ManagedAssignmentsDefinition{
						Selection: "topic-cluster-use",
						RackConstraints: PartitionRacks{
							{"zone-a", "zone-b"},
							{"zone-b", "zone-c"},
							{"zone-c"},
						},
					},
				},
			},
			wantErr: "number of replicas in a partition's rack constraints must match replication factor",
		},
		{
			name: "Tests invalid rack id",
			topicDef: TopicDefinition{
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions:        2,
					ReplicationFactor: 2,
					ManagedAssignments: &ManagedAssignmentsDefinition{
						Selection: "topic-cluster-use",
						RackConstraints: PartitionRacks{
							{"zone-a", "zone-b"},
							{"zone-b", ""},
						},
					},
				},
			},
			wantErr: "rack ids cannot be an empty string",
		},
		{
			name: "Tests specifying assignments and managed assignments together",
			topicDef: TopicDefinition{
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					Assignments: PartitionAssignments{
						{1, 2},
						{2, 3},
						{3, 1},
					},
					ManagedAssignments: &ManagedAssignmentsDefinition{
						RackConstraints: PartitionRacks{
							{"zone-a", "zone-b"},
							{"zone-b", "zone-c"},
							{"zone-c", "zone-a"},
						},
					},
				},
			},
			wantErr: "assignments and managed assignments cannot be specified together",
		},
		{
			name: "Tests a valid TopicDefinition",
			topicDef: TopicDefinition{
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					Assignments: PartitionAssignments{
						{1, 2},
						{2, 3},
						{3, 1},
					},
				},
			},
			wantErr: "",
		},
		{
			name: "Tests a valid TopicDefinition with managed assignments default",
			topicDef: TopicDefinition{
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
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
	resDef := ResourceDefinition{
		APIVersion: "v1",
		Kind:       "topic",
		Metadata: ResourceMetadataDefinition{
			Name: "foo",
		},
	}
	brokers := meta.Brokers{
		meta.Broker{ID: 1, Rack: "zone-a"},
		meta.Broker{ID: 2, Rack: "zone-a"},
		meta.Broker{ID: 3, Rack: "zone-b"},
		meta.Broker{ID: 4, Rack: "zone-b"},
		meta.Broker{ID: 5, Rack: "zone-c"},
		meta.Broker{ID: 6, Rack: "zone-c"},
	}

	type args struct {
		brokers meta.Brokers
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
				ResourceDefinition: resDef,
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
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions:        6,
					ReplicationFactor: 2,
					Assignments: PartitionAssignments{
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
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions:        6,
					ReplicationFactor: 2,
					ManagedAssignments: &ManagedAssignmentsDefinition{
						RackConstraints: PartitionRacks{
							{"zone-a", "zone-b"},
							{"zone-b", "zone-c"},
							{"zone-c", "zone-a"},
							{"zone-a", "zone-b"},
							{"zone-b", "zone-z"},
							{"zone-c", "zone-a"},
						},
					},
				},
			},
			args: args{
				brokers: brokers,
			},
			wantErr: "invalid rack id \"zone-z\" in rack constraints",
		},
		{
			name: "Tests when a rack ID is specified more times than available brokers",
			topicDef: TopicDefinition{
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 3,
					ManagedAssignments: &ManagedAssignmentsDefinition{
						RackConstraints: PartitionRacks{
							{"zone-a", "zone-b", "zone-a"},
							{"zone-b", "zone-c", "zone-b"},
							{"zone-c", "zone-c", "zone-c"},
						},
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
				ResourceDefinition: resDef,
				Spec: TopicSpecDefinition{
					Partitions:        3,
					ReplicationFactor: 2,
					Assignments: PartitionAssignments{
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
