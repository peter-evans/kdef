package compose_fixture

import (
	"reflect"
	"testing"
)

func TestComposeFixture_Env(t *testing.T) {
	type fields struct {
		ZookeeperPort int
		BrokerPort    int
		Brokers       int
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]string
	}{
		{
			name: "Tests return of env var map",
			fields: fields{
				ZookeeperPort: 2181,
				BrokerPort:    9092,
				Brokers:       6,
			},
			want: map[string]string{
				"ZOOKEEPER_PORT": "2181",
				"BROKER1_PORT":   "9092",
				"BROKER2_PORT":   "9093",
				"BROKER3_PORT":   "9094",
				"BROKER4_PORT":   "9095",
				"BROKER5_PORT":   "9096",
				"BROKER6_PORT":   "9097",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := ComposeFixture{
				ZookeeperPort: tt.fields.ZookeeperPort,
				BrokerPort:    tt.fields.BrokerPort,
				Brokers:       tt.fields.Brokers,
			}
			if got := tr.Env(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ComposeFixture.Env() = %v, want %v", got, tt.want)
			}
		})
	}
}
