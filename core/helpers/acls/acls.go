package acls

import (
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/util/str"
)

// Find the acl entry groups in A that are not in B (patch), and the common acl entry groups (intersection)
func DiffPatchIntersection(a def.AclEntryGroups, b def.AclEntryGroups) (def.AclEntryGroups, def.AclEntryGroups) {
	var patch def.AclEntryGroups
	var intersection def.AclEntryGroups
	for _, group := range a {
		for _, principle := range group.Principals {
			for _, host := range group.Hosts {
				for _, operation := range group.Operations {
					entryGroup := def.AclEntryGroup{
						Principals:     []string{principle},
						Hosts:          []string{host},
						Operations:     []string{operation},
						PermissionType: group.PermissionType,
					}
					if b.Contains(
						principle,
						host,
						operation,
						group.PermissionType,
					) {
						intersection = append(intersection, entryGroup)
					} else {
						patch = append(patch, entryGroup)
					}
				}
			}
		}
	}
	return patch, intersection
}

// Merge acl entry groups
func MergeGroups(groups def.AclEntryGroups) def.AclEntryGroups {
	count := len(groups)
	if count == 1 {
		return groups
	}

	// Call recursively until the groups cannot be merged further
	mergedGroups := mergeGroups(groups[0:1], groups[1:])
	if len(mergedGroups) < count {
		return MergeGroups(mergedGroups)
	} else {
		return mergedGroups
	}
}

// Try to merge group A with group B
func mergeGroups(a def.AclEntryGroups, b def.AclEntryGroups) def.AclEntryGroups {
	// Loop through the elements of A trying to merge them with the first element of B
	var groups def.AclEntryGroups
	var merged bool
	for _, ag := range a {
		if !merged {
			if merge := tryMergeGroups(ag, b[0]); len(merge) == 1 {
				groups = append(groups, merge...)
				merged = true
				continue
			}
		}
		groups = append(groups, ag)
	}
	// If the first element of B couldn't merge with any element of A then just append
	if !merged {
		groups = append(groups, b[0])
	}

	// Call this function recursively if B contains further elements
	if len(b) > 1 {
		return mergeGroups(groups, b[1:])
	} else {
		return groups
	}
}

// Try to merge acl entry groups, returning the merged group or unmerged input groups
func tryMergeGroups(a def.AclEntryGroup, b def.AclEntryGroup) def.AclEntryGroups {
	if a.PermissionType == b.PermissionType {
		// Two out of the following three properties must match to be merged
		equalPrincipals := str.UnorderedEqual(a.Principals, b.Principals)
		equalHosts := str.UnorderedEqual(a.Hosts, b.Hosts)
		equalOperations := str.UnorderedEqual(a.Operations, b.Operations)

		if equalPrincipals && equalHosts {
			var operations []string
			operations = append(operations, a.Operations...)
			operations = append(operations, b.Operations...)
			operations = str.Deduplicate(operations)
			return def.AclEntryGroups{
				def.AclEntryGroup{
					Principals:     a.Principals,
					Hosts:          a.Hosts,
					Operations:     operations,
					PermissionType: a.PermissionType,
				},
			}
		} else if equalPrincipals && equalOperations {
			var hosts []string
			hosts = append(hosts, a.Hosts...)
			hosts = append(hosts, b.Hosts...)
			hosts = str.Deduplicate(hosts)
			return def.AclEntryGroups{
				def.AclEntryGroup{
					Principals:     a.Principals,
					Hosts:          hosts,
					Operations:     a.Operations,
					PermissionType: a.PermissionType,
				},
			}
		} else if equalHosts && equalOperations {
			var principals []string
			principals = append(principals, a.Principals...)
			principals = append(principals, b.Principals...)
			principals = str.Deduplicate(principals)
			return def.AclEntryGroups{
				def.AclEntryGroup{
					Principals:     principals,
					Hosts:          a.Hosts,
					Operations:     a.Operations,
					PermissionType: a.PermissionType,
				},
			}
		}
	}

	return def.AclEntryGroups{a, b}
}
