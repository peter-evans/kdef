// Package def implements definitions for Kafka resources.
package def

import (
	"testing"

	"github.com/peter-evans/kdef/core/test/tutil"
)

func TestACLDefinition_Validate(t *testing.T) {
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
					Kind:       KindACL,
					Metadata: ResourceMetadataDefinition{
						Name:                "foo",
						ResourcePatternType: "literal",
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
					Kind:       KindACL,
					Metadata: ResourceMetadataDefinition{
						Name:                "foo",
						Type:                "bar",
						ResourcePatternType: "literal",
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
					Kind:       KindACL,
					Metadata: ResourceMetadataDefinition{
						Name:                "foo",
						Type:                "cluster",
						ResourcePatternType: "literal",
					},
				},
			},
			wantErr: "metadata name must be \"kafka-cluster\" when type is \"cluster\"",
		},
		{
			name: "Tests missing metadata resource pattern type",
			aclDef: ACLDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       KindACL,
					Metadata: ResourceMetadataDefinition{
						Name: "foo",
						Type: "topic",
					},
				},
			},
			wantErr: "metadata resource pattern type must be supplied",
		},
		{
			name: "Tests invalid metadata resource pattern type",
			aclDef: ACLDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       KindACL,
					Metadata: ResourceMetadataDefinition{
						Name:                "foo",
						Type:                "topic",
						ResourcePatternType: "bar",
					},
				},
			},
			wantErr: "metadata resource pattern type must be one of \"literal|prefixed\"",
		},
		{
			name: "Tests missing acl principals",
			aclDef: ACLDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       KindACL,
					Metadata: ResourceMetadataDefinition{
						Name:                "foo",
						Type:                "topic",
						ResourcePatternType: "literal",
					},
				},
				Spec: ACLSpecDefinition{
					ACLs: ACLEntryGroups{
						ACLEntryGroup{
							Hosts:          []string{"*"},
							Operations:     []string{"READ"},
							PermissionType: "ALLOW",
						},
					},
				},
			},
			wantErr: "principals are missing from acl entry group",
		},
		{
			name: "Tests missing acl hosts",
			aclDef: ACLDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       KindACL,
					Metadata: ResourceMetadataDefinition{
						Name:                "foo",
						Type:                "topic",
						ResourcePatternType: "literal",
					},
				},
				Spec: ACLSpecDefinition{
					ACLs: ACLEntryGroups{
						ACLEntryGroup{
							Principals:     []string{"User:foo"},
							Operations:     []string{"READ"},
							PermissionType: "ALLOW",
						},
					},
				},
			},
			wantErr: "hosts are missing from acl entry group",
		},
		{
			name: "Tests missing acl operations",
			aclDef: ACLDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       KindACL,
					Metadata: ResourceMetadataDefinition{
						Name:                "foo",
						Type:                "topic",
						ResourcePatternType: "literal",
					},
				},
				Spec: ACLSpecDefinition{
					ACLs: ACLEntryGroups{
						ACLEntryGroup{
							Principals:     []string{"User:foo"},
							Hosts:          []string{"*"},
							PermissionType: "ALLOW",
						},
					},
				},
			},
			wantErr: "operations are missing from acl entry group",
		},
		{
			name: "Tests invalid acl operation",
			aclDef: ACLDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       KindACL,
					Metadata: ResourceMetadataDefinition{
						Name:                "foo",
						Type:                "topic",
						ResourcePatternType: "literal",
					},
				},
				Spec: ACLSpecDefinition{
					ACLs: ACLEntryGroups{
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
					Kind:       KindACL,
					Metadata: ResourceMetadataDefinition{
						Name:                "foo",
						Type:                "topic",
						ResourcePatternType: "literal",
					},
				},
				Spec: ACLSpecDefinition{
					ACLs: ACLEntryGroups{
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
			name: "Tests a valid ACLDefinition",
			aclDef: ACLDefinition{
				ResourceDefinition: ResourceDefinition{
					APIVersion: "v1",
					Kind:       KindACL,
					Metadata: ResourceMetadataDefinition{
						Name:                "foo",
						Type:                "topic",
						ResourcePatternType: "literal",
					},
				},
				Spec: ACLSpecDefinition{
					ACLs: ACLEntryGroups{
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
				t.Errorf("ACLDefinition.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
