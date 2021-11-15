// Package def implements definitions for Kafka resources.
package def

import (
	"fmt"
	"strings"

	"github.com/bradfitz/slice" //nolint
	"github.com/gotidy/copy"
	"github.com/peter-evans/kdef/core/util/str"
)

var aclResourceTypes = []string{
	"topic",
	"group",
	"cluster",
	"transactional_id",
	"delegation_token",
}

var aclOperations = []string{
	"ALL",
	"READ",
	"WRITE",
	"CREATE",
	"DELETE",
	"ALTER",
	"DESCRIBE",
	"CLUSTER_ACTION",
	"DESCRIBE_CONFIGS",
	"ALTER_CONFIGS",
	"IDEMPOTENT_WRITE",
}

var aclPermissionTypes = []string{"ALLOW", "DENY"}

// ACLEntryGroup represents an ACL entry group.
type ACLEntryGroup struct {
	Principals     []string `json:"principals"`
	Hosts          []string `json:"hosts"`
	Operations     []string `json:"operations"`
	PermissionType string   `json:"permissionType"`
}

// ACLEntryGroups represents a slice of ACL entry groups.
type ACLEntryGroups []ACLEntryGroup

// Validate validates ACL entry groups.
func (a ACLEntryGroups) Validate() error {
	for _, group := range a {
		for _, operation := range group.Operations {
			if !str.Contains(operation, aclOperations) {
				return fmt.Errorf("acl operation must be one of %q", strings.Join(aclOperations, "|"))
			}
		}
		if !str.Contains(group.PermissionType, aclPermissionTypes) {
			return fmt.Errorf("acl permission type must be one of %q", strings.Join(aclPermissionTypes, "|"))
		}
	}

	return nil
}

// Contains determines if an ACL entry is contained in any group.
func (a ACLEntryGroups) Contains(
	principal string,
	host string,
	operation string,
	permissionType string,
) bool {
	for _, group := range a {
		if str.Contains(principal, group.Principals) &&
			str.Contains(host, group.Hosts) &&
			str.Contains(operation, group.Operations) &&
			group.PermissionType == permissionType {
			return true
		}
	}
	return false
}

// Sort sorts ACL entry groups.
func (a ACLEntryGroups) Sort() {
	// TODO: Use sort.Slice in the standard library after upgrading to Go 1.8.
	//nolint
	slice.Sort(a[:], func(i, j int) bool {
		return a[i].Principals[0] < a[j].Principals[0] ||
			a[i].Principals[0] == a[j].Principals[0] &&
				a[i].Hosts[0] < a[j].Hosts[0] ||
			a[i].Principals[0] == a[j].Principals[0] &&
				a[i].Hosts[0] == a[j].Hosts[0] &&
				a[i].Operations[0] < a[j].Operations[0] ||
			a[i].Principals[0] == a[j].Principals[0] &&
				a[i].Hosts[0] == a[j].Hosts[0] &&
				a[i].Operations[0] == a[j].Operations[0] &&
				a[i].PermissionType < a[j].PermissionType
	})
}

// ACLSpecDefinition represents an ACL spec definition.
type ACLSpecDefinition struct {
	ACLs                ACLEntryGroups `json:"acls,omitempty"`
	DeleteUndefinedACLs bool           `json:"deleteUndefinedAcls"`
}

// ACLDefinition represents an ACL resource definition.
type ACLDefinition struct {
	ResourceDefinition
	Spec ACLSpecDefinition `json:"spec"`
}

// Copy creates a copy of this ACLDefinition.
func (a ACLDefinition) Copy() ACLDefinition {
	copiers := copy.New()
	copier := copiers.Get(&ACLDefinition{}, &ACLDefinition{})
	var aclDefCopy ACLDefinition
	copier.Copy(&aclDefCopy, &a)
	return aclDefCopy
}

// Validate validates the definition.
func (a ACLDefinition) Validate() error {
	if err := a.ValidateResource(); err != nil {
		return err
	}

	if len(a.ResourceDefinition.Metadata.Type) == 0 {
		return fmt.Errorf("metadata type must be supplied")
	}

	if !str.Contains(a.ResourceDefinition.Metadata.Type, aclResourceTypes) {
		return fmt.Errorf("metadata type must be one of %q", strings.Join(aclResourceTypes, "|"))
	}

	if a.ResourceDefinition.Metadata.Type == "cluster" && a.ResourceDefinition.Metadata.Name != "kafka-cluster" {
		return fmt.Errorf("metadata name must be \"kafka-cluster\" when type is \"cluster\"")
	}

	if err := a.Spec.ACLs.Validate(); err != nil {
		return err
	}

	return nil
}

// NewACLDefinition creates an ACL definition from metadata and config.
func NewACLDefinition(
	name string,
	resourceType string,
	acls ACLEntryGroups,
) ACLDefinition {
	aclDef := ACLDefinition{
		ResourceDefinition: ResourceDefinition{
			APIVersion: "v1",
			Kind:       "acl",
			Metadata: ResourceMetadataDefinition{
				Name: name,
				Type: resourceType,
			},
		},
		Spec: ACLSpecDefinition{
			ACLs: acls,
		},
	}

	return aclDef
}
