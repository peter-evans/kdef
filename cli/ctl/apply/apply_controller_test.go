// Package apply implements the apply controller.
package apply

import (
	"reflect"
	"testing"

	"github.com/peter-evans/kdef/cli/test/tutil"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/opt"
)

func Test_getResourceDefinitions(t *testing.T) {
	type args struct {
		defDocs []string
		format  opt.DefinitionFormat
	}
	tests := []struct {
		name    string
		args    args
		want    []def.ResourceDefinition
		wantErr string
	}{
		{
			name: "Tests invalid doc (YAML)",
			args: args{
				defDocs: []string{"foo"},
				format:  opt.YAMLFormat,
			},
			want:    nil,
			wantErr: "error unmarshaling JSON",
		},
		{
			name: "Tests invalid kind (YAML)",
			args: args{
				defDocs: []string{"apiVersion: v1\nkind: foo"},
				format:  opt.YAMLFormat,
			},
			want:    nil,
			wantErr: "invalid definition kind",
		},
		{
			name: "Tests invalid apiVersion (YAML)",
			args: args{
				defDocs: []string{"apiVersion: foo\nkind: topic"},
				format:  opt.YAMLFormat,
			},
			want:    nil,
			wantErr: "invalid definition apiVersion",
		},
		{
			name: "Tests return of resource definitions (YAML)",
			args: args{
				defDocs: []string{
					"apiVersion: v1\nkind: acl\nmetadata:\n  name: \"topic_foo\"\n  type: \"topic\"\n  resourcePatternType: \"literal\"",
					"apiVersion: v1\nkind: broker\nmetadata:\n  name: \"1\"",
					"apiVersion: v1\nkind: brokers\nmetadata:\n  name: brokers_foo",
					"apiVersion: v1\nkind: topic\nmetadata:\n  name: topic_foo",
				},
				format: opt.YAMLFormat,
			},
			want: []def.ResourceDefinition{
				{
					APIVersion: "v1",
					Kind:       def.KindACL,
					Metadata: def.ResourceMetadataDefinition{
						Name:                "topic_foo",
						Type:                "topic",
						ResourcePatternType: "literal",
					},
				},
				{
					APIVersion: "v1",
					Kind:       def.KindBroker,
					Metadata: def.ResourceMetadataDefinition{
						Name: "1",
					},
				},
				{
					APIVersion: "v1",
					Kind:       def.KindBrokers,
					Metadata: def.ResourceMetadataDefinition{
						Name: "brokers_foo",
					},
				},
				{
					APIVersion: "v1",
					Kind:       def.KindTopic,
					Metadata: def.ResourceMetadataDefinition{
						Name: "topic_foo",
					},
				},
			},
			wantErr: "",
		},
		{
			name: "Tests invalid doc (JSON)",
			args: args{
				defDocs: []string{"invalid json"},
				format:  opt.JSONFormat,
			},
			want:    nil,
			wantErr: "invalid character 'i' looking for beginning of value",
		},
		{
			name: "Tests invalid kind (JSON)",
			args: args{
				defDocs: []string{"{\"apiVersion\": \"v1\", \"kind\": \"foo\"}"},
				format:  opt.JSONFormat,
			},
			want:    nil,
			wantErr: "invalid definition kind",
		},
		{
			name: "Tests invalid apiVersion (JSON)",
			args: args{
				defDocs: []string{"{\"apiVersion\": \"foo\", \"kind\": \"topic\"}"},
				format:  opt.JSONFormat,
			},
			want:    nil,
			wantErr: "invalid definition apiVersion",
		},
		{
			name: "Tests return of resource definitions (JSON)",
			args: args{
				defDocs: []string{
					"{\"apiVersion\": \"v1\", \"kind\": \"acl\", \"metadata\": {\"name\": \"topic_foo\", \"type\": \"topic\", \"resourcePatternType\": \"literal\"}}",
					"{\"apiVersion\": \"v1\", \"kind\": \"broker\", \"metadata\": {\"name\": \"1\"}}",
					"{\"apiVersion\": \"v1\", \"kind\": \"brokers\", \"metadata\": {\"name\": \"brokers_foo\"}}",
					"{\"apiVersion\": \"v1\", \"kind\": \"topic\", \"metadata\": {\"name\": \"topic_foo\"}}",
				},
				format: opt.YAMLFormat,
			},
			want: []def.ResourceDefinition{
				{
					APIVersion: "v1",
					Kind:       def.KindACL,
					Metadata: def.ResourceMetadataDefinition{
						Name:                "topic_foo",
						Type:                "topic",
						ResourcePatternType: "literal",
					},
				},
				{
					APIVersion: "v1",
					Kind:       def.KindBroker,
					Metadata: def.ResourceMetadataDefinition{
						Name: "1",
					},
				},
				{
					APIVersion: "v1",
					Kind:       def.KindBrokers,
					Metadata: def.ResourceMetadataDefinition{
						Name: "brokers_foo",
					},
				},
				{
					APIVersion: "v1",
					Kind:       def.KindTopic,
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
			got, err := getResourceDefinitions(tt.args.defDocs, tt.args.format)
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
