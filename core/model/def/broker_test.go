// Package def implements definitions for Kafka resources.
package def

import (
	"testing"

	"github.com/peter-evans/kdef/core/model/meta"
	"github.com/peter-evans/kdef/core/test/tutil"
)

func TestBrokerDefinition_Validate(t *testing.T) {
	tests := []struct {
		name      string
		brokerDef BrokerDefinition
		wantErr   string
	}{
		{
			name: "Tests an invalid metadata name",
			brokerDef: BrokerDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       KindBroker,
					Metadata: ResourceMetadataDefinition{
						Name: "foo",
					},
				},
			},
			wantErr: "metadata name must be an integer broker id",
		},
		{
			name: "Tests a valid broker definition",
			brokerDef: BrokerDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       KindBroker,
					Metadata: ResourceMetadataDefinition{
						Name: "1",
					},
				},
			},
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.brokerDef.Validate(); !tutil.ErrorContains(err, tt.wantErr) {
				t.Errorf("BrokerDefinition.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBrokerDefinition_ValidateWithMetadata(t *testing.T) {
	brokers := meta.Brokers{
		meta.Broker{ID: 1, Rack: "zone-a"},
		meta.Broker{ID: 2, Rack: "zone-b"},
		meta.Broker{ID: 3, Rack: "zone-c"},
	}

	type args struct {
		brokers meta.Brokers
	}
	tests := []struct {
		name      string
		brokerDef BrokerDefinition
		args      args
		wantErr   string
	}{
		{
			name: "Tests an invalid metadata name",
			brokerDef: BrokerDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       KindBroker,
					Metadata: ResourceMetadataDefinition{
						Name: "9",
					},
				},
			},
			args: args{
				brokers: brokers,
			},
			wantErr: "metadata name must be the id of an available broker",
		},
		{
			name: "Tests a valid broker definition",
			brokerDef: BrokerDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       KindBroker,
					Metadata: ResourceMetadataDefinition{
						Name: "1",
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
			if err := tt.brokerDef.ValidateWithMetadata(tt.args.brokers); !tutil.ErrorContains(err, tt.wantErr) {
				t.Errorf("BrokerDefinition.ValidateWithMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
