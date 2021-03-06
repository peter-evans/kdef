// Package meta implements metadata structures and related operations.
package meta

import (
	"reflect"
	"testing"
)

func TestBrokers_IDs(t *testing.T) {
	tests := []struct {
		name string
		b    Brokers
		want []int32
	}{
		{
			name: "Test return of a broker ID slice",
			b: Brokers{
				Broker{ID: 1, Rack: "zone-a"},
				Broker{ID: 2, Rack: "zone-b"},
				Broker{ID: 3, Rack: "zone-c"},
			},
			want: []int32{1, 2, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.b.IDs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Brokers.IDs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBrokers_BrokersByRack(t *testing.T) {
	tests := []struct {
		name string
		b    Brokers
		want map[string][]int32
	}{
		{
			name: "Test return of brokers by rack ID",
			b: Brokers{
				Broker{ID: 1, Rack: "zone-a"},
				Broker{ID: 2, Rack: "zone-b"},
				Broker{ID: 3, Rack: "zone-c"},
				Broker{ID: 4, Rack: "zone-a"},
				Broker{ID: 5, Rack: ""}, // Rack id not set on broker
				Broker{ID: 6, Rack: "zone-c"},
			},
			want: map[string][]int32{
				"zone-a": {1, 4},
				"zone-b": {2},
				"zone-c": {3, 6},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.b.BrokersByRack(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Brokers.BrokersByRack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBrokers_Racks(t *testing.T) {
	tests := []struct {
		name string
		b    Brokers
		want []string
	}{
		{
			name: "Test return of a unique rack ID slice",
			b: Brokers{
				Broker{ID: 1, Rack: "zone-a"},
				Broker{ID: 2, Rack: "zone-b"},
				Broker{ID: 3, Rack: "zone-c"},
				Broker{ID: 4, Rack: "zone-a"},
				Broker{ID: 5, Rack: ""}, // Rack id not set on broker
				Broker{ID: 6, Rack: "zone-c"},
			},
			want: []string{"zone-a", "zone-b", "zone-c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.b.Racks(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Brokers.Racks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBrokers_RacksByBroker(t *testing.T) {
	tests := []struct {
		name string
		b    Brokers
		want map[int32]string
	}{
		{
			name: "Test return of racks by broker ID",
			b: Brokers{
				Broker{ID: 1, Rack: "zone-a"},
				Broker{ID: 2, Rack: "zone-b"},
				Broker{ID: 3, Rack: "zone-c"},
				Broker{ID: 4, Rack: "zone-a"},
				Broker{ID: 5, Rack: ""}, // Rack id not set on broker
				Broker{ID: 6, Rack: "zone-c"},
			},
			want: map[int32]string{
				1: "zone-a",
				2: "zone-b",
				3: "zone-c",
				4: "zone-a",
				5: "",
				6: "zone-c",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.b.RacksByBroker(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Brokers.RacksByBroker() = %v, want %v", got, tt.want)
			}
		})
	}
}
