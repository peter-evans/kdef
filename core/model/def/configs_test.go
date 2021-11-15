// Package def implements definitions for Kafka resources.
package def

import (
	"reflect"
	"testing"
)

func TestConfigKey_IsDynamic(t *testing.T) {
	tests := []struct {
		name       string
		configItem ConfigKey
		want       bool
	}{
		{
			name: "Tests a dynamic key",
			configItem: ConfigKey{
				Source: ConfigSourceDynamicBrokerConfig,
			},
			want: true,
		},
		{
			name: "Tests a non-dynamic key",
			configItem: ConfigKey{
				Source: ConfigSourceDefaultConfig,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.configItem.IsDynamic(); got != tt.want {
				t.Errorf("ConfigKey.IsDynamic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigs_ToMap(t *testing.T) {
	v1 := "foo-value"
	v2 := "bar-value"
	v3 := "baz-value"

	tests := []struct {
		name string
		c    Configs
		want ConfigsMap
	}{
		{
			name: "Tests return of a config map",
			c: Configs{
				ConfigKey{Name: "foo", Value: &v1},
				ConfigKey{Name: "bar", Value: &v2},
				ConfigKey{Name: "baz", Value: &v3},
			},
			want: ConfigsMap{
				"foo": &v1,
				"bar": &v2,
				"baz": &v3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.ToMap(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Configs.ToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigs_ToExportableMap(t *testing.T) {
	v1 := "foo-value"
	v2 := "bar-value"
	v3 := "baz-value"

	tests := []struct {
		name string
		c    Configs
		want ConfigsMap
	}{
		{
			name: "Tests return of a config map",
			c: Configs{
				ConfigKey{Name: "foo", Value: &v1},
				ConfigKey{Name: "bar", Value: &v2, IsSensitive: true},
				ConfigKey{Name: "baz", Value: &v3},
			},
			want: ConfigsMap{
				"foo": &v1,
				"baz": &v3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.ToExportableMap(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Configs.ToExportableMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
