package apply

import (
	"reflect"
	"testing"

	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/test/tutil"
)

func Test_getResourceDefinitions(t *testing.T) {
	type args struct {
		yamlDocs []string
	}
	tests := []struct {
		name    string
		args    args
		want    []def.ResourceDefinition
		wantErr string
	}{
		{
			name: "Tests invalid YAML doc",
			args: args{
				yamlDocs: []string{"foo"},
			},
			want:    nil,
			wantErr: "error unmarshaling JSON",
		},
		{
			name: "Tests invalid kind",
			args: args{
				yamlDocs: []string{"apiVersion: v1\nkind: foo"},
			},
			want:    nil,
			wantErr: "invalid definition kind",
		},
		{
			name: "Tests invalid apiVersion",
			args: args{
				yamlDocs: []string{"apiVersion: foo\nkind: topic"},
			},
			want:    nil,
			wantErr: "invalid definition apiVersion",
		},
		{
			// TODO: add other resources
			name: "Tests return of resource definitions",
			args: args{
				yamlDocs: []string{
					"apiVersion: v1\nkind: broker\nmetadata:\n  name: \"1\"",
					"apiVersion: v1\nkind: brokers\nmetadata:\n  name: brokers_foo",
					"apiVersion: v1\nkind: topic\nmetadata:\n  name: topic_foo",
				},
			},
			want: []def.ResourceDefinition{
				{
					ApiVersion: "v1",
					Kind:       "broker",
					Metadata: def.ResourceMetadataDefinition{
						Name: "1",
					},
				},
				{
					ApiVersion: "v1",
					Kind:       "brokers",
					Metadata: def.ResourceMetadataDefinition{
						Name: "brokers_foo",
					},
				},
				{
					ApiVersion: "v1",
					Kind:       "topic",
					Metadata: def.ResourceMetadataDefinition{
						Name: "topic_foo",
					},
				},
			},
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getResourceDefinitions(tt.args.yamlDocs)
			if !tutil.ErrorContains(err, tt.wantErr) {
				t.Errorf("getResourceDefinitions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getResourceDefinitions() = %v, want %v", got, tt.want)
			}
		})
	}
}
