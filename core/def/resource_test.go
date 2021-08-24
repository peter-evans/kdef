package def

import (
	"reflect"
	"testing"

	"github.com/peter-evans/kdef/test/tutil"
)

func TestGetResourceDefinitions(t *testing.T) {
	type args struct {
		yamlDocs []string
	}
	tests := []struct {
		name    string
		args    args
		want    []ResourceDefinition
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
			name: "Tests return of resource definitions",
			args: args{
				yamlDocs: []string{"apiVersion: v1\nkind: topic"},
			},
			want: []ResourceDefinition{
				{
					ApiVersion: "v1",
					Kind:       "topic",
				},
			},
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetResourceDefinitions(tt.args.yamlDocs)
			if !tutil.ErrorContains(err, tt.wantErr) {
				t.Errorf("GetResourceDefinitions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetResourceDefinitions() = %v, want %v", got, tt.want)
			}
		})
	}
}
