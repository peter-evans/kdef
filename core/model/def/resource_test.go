package def

import (
	"testing"

	"github.com/peter-evans/kdef/core/test/tutil"
)

func TestResourceDefinition_ValidateResource(t *testing.T) {
	type fields struct {
		ApiVersion string
		Kind       string
		Metadata   ResourceMetadataDefinition
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr string
	}{
		{
			name: "Tests invalid apiVersion",
			fields: fields{
				ApiVersion: "foo",
				Kind:       "topic",
				Metadata: ResourceMetadataDefinition{
					Name: "foo",
				},
			},
			wantErr: "invalid definition apiVersion \"foo\"",
		},
		{
			name: "Tests invalid kind",
			fields: fields{
				ApiVersion: "v1",
				Kind:       "foo",
				Metadata: ResourceMetadataDefinition{
					Name: "foo",
				},
			},
			wantErr: "invalid definition kind",
		},
		{
			name: "Tests missing metadata name",
			fields: fields{
				ApiVersion: "v1",
				Kind:       "topic",
			},
			wantErr: "metadata name must be supplied",
		},
		{
			name: "Tests valid resource definition",
			fields: fields{
				ApiVersion: "v1",
				Kind:       "topic",
				Metadata: ResourceMetadataDefinition{
					Name: "foo",
				},
			},
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ResourceDefinition{
				ApiVersion: tt.fields.ApiVersion,
				Kind:       tt.fields.Kind,
				Metadata:   tt.fields.Metadata,
			}
			err := r.ValidateResource()
			if !tutil.ErrorContains(err, tt.wantErr) {
				t.Errorf("ResourceDefinition.ValidateResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
