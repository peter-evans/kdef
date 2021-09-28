package def

import (
	"reflect"
	"testing"
)

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
				ConfigItem{Name: "foo", Value: &v1},
				ConfigItem{Name: "bar", Value: &v2},
				ConfigItem{Name: "baz", Value: &v3},
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
