package def

import (
	"testing"

	"github.com/peter-evans/kdef/core/test/tutil"
)

func TestAclDefinition_Validate(t *testing.T) {
	tests := []struct {
		name    string
		aclDef  ACLDefinition
		wantErr string
	}{
		{
			name: "Tests missing metadata resource type",
			aclDef: ACLDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       "acl",
					Metadata: ResourceMetadataDefinition{
						Name: "foo",
					},
				},
			},
			wantErr: "metadata type must be supplied",
		},
		{
			name: "Tests invalid metadata resource type",
			aclDef: ACLDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       "acl",
					Metadata: ResourceMetadataDefinition{
						Name: "foo",
						Type: "bar",
					},
				},
			},
			wantErr: "metadata type must be one of \"topic|group|cluster|transactional_id|delegation_token\"",
		},
		{
			name: "Tests invalid metadata name when type is cluster",
			aclDef: ACLDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       "acl",
					Metadata: ResourceMetadataDefinition{
						Name: "foo",
						Type: "cluster",
					},
				},
			},
			wantErr: "metadata name must be \"kafka-cluster\" when type is \"cluster\"",
		},
		{
			name: "Tests invalid acl operation",
			aclDef: ACLDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       "acl",
					Metadata: ResourceMetadataDefinition{
						Name: "foo",
						Type: "topic",
					},
				},
				Spec: ACLSpecDefinition{
					Acls: ACLEntryGroups{
						ACLEntryGroup{
							Principals:     []string{"User:foo"},
							Hosts:          []string{"*"},
							Operations:     []string{"BAR"},
							PermissionType: "ALLOW",
						},
					},
				},
			},
			wantErr: "acl operation must be one of",
		},
		{
			name: "Tests invalid acl permission type",
			aclDef: ACLDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       "acl",
					Metadata: ResourceMetadataDefinition{
						Name: "foo",
						Type: "topic",
					},
				},
				Spec: ACLSpecDefinition{
					Acls: ACLEntryGroups{
						ACLEntryGroup{
							Principals:     []string{"User:foo"},
							Hosts:          []string{"*"},
							Operations:     []string{"READ"},
							PermissionType: "BAR",
						},
					},
				},
			},
			wantErr: "acl permission type must be one of \"ALLOW|DENY\"",
		},
		{
			name: "Tests a valid AclDefinition",
			aclDef: ACLDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       "acl",
					Metadata: ResourceMetadataDefinition{
						Name: "foo",
						Type: "topic",
					},
				},
				Spec: ACLSpecDefinition{
					Acls: ACLEntryGroups{
						ACLEntryGroup{
							Principals:     []string{"User:foo"},
							Hosts:          []string{"*"},
							Operations:     []string{"READ", "WRITE", "CREATE"},
							PermissionType: "ALLOW",
						},
					},
				},
			},
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.aclDef.Validate(); !tutil.ErrorContains(err, tt.wantErr) {
				t.Errorf("AclDefinition.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
