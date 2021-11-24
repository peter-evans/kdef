// Package kafka implements the Kafka service handling requests and responses.
package kafka

import (
	"context"
	"fmt"
	"strings"

	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// ResourceACLs represents ACLs for a named resource
type ResourceACLs struct {
	ResourceName string
	ResourceType string
	ACLs         def.ACLEntryGroups
}

// describeResourceACLs executes a request to describe ACLs of a specific resource (Kafka 0.11.0+).
func describeResourceACLs(
	cl *client.Client,
	name string,
	resourceType string,
) (def.ACLEntryGroups, error) {
	resType, err := kmsg.ParseACLResourceType(resourceType)
	if err != nil {
		return nil, err
	}

	req := kmsg.NewDescribeACLsRequest()
	req.ResourceName = &name
	req.ResourceType = resType
	req.Operation = kmsg.ACLOperationAny
	req.PermissionType = kmsg.ACLPermissionTypeAny
	req.ResourcePatternType = kmsg.ACLResourcePatternTypeLiteral

	resourceACLs, err := describeACLs(cl, req)
	if err != nil {
		return nil, err
	}

	if len(resourceACLs) > 0 {
		return resourceACLs[0].ACLs, nil
	}

	return nil, nil
}

// describeAllResourceACLs executes a request to describe ACLs for all resources (Kafka 0.11.0+).
func describeAllResourceACLs(
	cl *client.Client,
	resourceType string,
) ([]ResourceACLs, error) {
	resType, err := kmsg.ParseACLResourceType(resourceType)
	if err != nil {
		return nil, err
	}

	req := kmsg.NewDescribeACLsRequest()
	req.ResourceType = resType
	req.Operation = kmsg.ACLOperationAny
	req.PermissionType = kmsg.ACLPermissionTypeAny
	req.ResourcePatternType = kmsg.ACLResourcePatternTypeAny

	return describeACLs(cl, req)
}

// describeACLs executes a request to describe resource ACLs (Kafka 0.11.0+).
func describeACLs(
	cl *client.Client,
	req kmsg.DescribeACLsRequest,
) ([]ResourceACLs, error) {
	kresp, err := cl.Client.Request(context.Background(), &req)
	if err != nil {
		return nil, err
	}
	resp := kresp.(*kmsg.DescribeACLsResponse)

	if err := kerr.ErrorForCode(resp.ErrorCode); err != nil {
		errMsg := err.Error()
		if resp.ErrorMessage != nil {
			errMsg = fmt.Sprintf("%s: %s", errMsg, *resp.ErrorMessage)
		}
		return nil, fmt.Errorf(errMsg)
	}

	resourceACLs := make([]ResourceACLs, len(resp.Resources))
	for i, resource := range resp.Resources {
		var acls def.ACLEntryGroups
		for _, acl := range resource.ACLs {
			acls = append(acls, def.ACLEntryGroup{
				Principals:     []string{acl.Principal},
				Hosts:          []string{acl.Host},
				Operations:     []string{acl.Operation.String()},
				PermissionType: acl.PermissionType.String(),
			})
		}

		if err := acls.Validate(); err != nil {
			return nil, err
		}

		acls.Sort()

		resourceACLs[i] = ResourceACLs{
			ResourceName: resource.ResourceName,
			ResourceType: strings.ToLower(resource.ResourceType.String()),
			ACLs:         acls,
		}
	}

	return resourceACLs, nil
}

// createACLs executes a request to create ACLs (Kafka 0.11.0+).
func createACLs(
	cl *client.Client,
	name string,
	resourceType string,
	acls def.ACLEntryGroups,
) error {
	resType, err := kmsg.ParseACLResourceType(resourceType)
	if err != nil {
		return err
	}

	var creations []kmsg.CreateACLsRequestCreation
	for _, group := range acls {
		permType, err := kmsg.ParseACLPermissionType(group.PermissionType)
		if err != nil {
			return err
		}
		for _, principal := range group.Principals {
			for _, host := range group.Hosts {
				for _, operation := range group.Operations {
					op, err := kmsg.ParseACLOperation(operation)
					if err != nil {
						return err
					}

					c := kmsg.NewCreateACLsRequestCreation()
					c.ResourceName = name
					c.ResourceType = resType
					c.ResourcePatternType = kmsg.ACLResourcePatternTypeLiteral
					c.Principal = principal
					c.Host = host
					c.Operation = op
					c.PermissionType = permType
					creations = append(creations, c)
				}
			}
		}
	}

	req := kmsg.NewCreateACLsRequest()
	req.Creations = creations

	kresp, err := cl.Client.Request(context.Background(), &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.CreateACLsResponse)

	if len(resp.Results) != len(creations) {
		return fmt.Errorf("requested creation of %d acls but received %d", len(creations), len(resp.Results))
	}

	for _, result := range resp.Results {
		if err := kerr.ErrorForCode(result.ErrorCode); err != nil {
			errMsg := err.Error()
			if result.ErrorMessage != nil {
				errMsg = fmt.Sprintf("%s: %s", errMsg, *result.ErrorMessage)
			}
			return fmt.Errorf(errMsg)
		}
	}

	return nil
}

// deleteACLs executes a request to delete ACLs (Kafka 0.11.0+).
func deleteACLs(
	cl *client.Client,
	name string,
	resourceType string,
	acls def.ACLEntryGroups,
) error {
	resType, err := kmsg.ParseACLResourceType(resourceType)
	if err != nil {
		return err
	}

	var filters []kmsg.DeleteACLsRequestFilter
	for _, group := range acls {
		permType, err := kmsg.ParseACLPermissionType(group.PermissionType)
		if err != nil {
			return err
		}
		for _, principal := range group.Principals {
			for _, host := range group.Hosts {
				for _, operation := range group.Operations {
					op, err := kmsg.ParseACLOperation(operation)
					if err != nil {
						return err
					}

					c := kmsg.NewDeleteACLsRequestFilter()
					c.ResourceName = &name
					c.ResourceType = resType
					c.ResourcePatternType = kmsg.ACLResourcePatternTypeLiteral
					c.Principal = &principal
					c.Host = &host
					c.Operation = op
					c.PermissionType = permType
					filters = append(filters, c)
				}
			}
		}
	}

	req := kmsg.NewDeleteACLsRequest()
	req.Filters = filters

	kresp, err := cl.Client.Request(context.Background(), &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.DeleteACLsResponse)

	if len(resp.Results) != len(filters) {
		return fmt.Errorf("requested deletion of %d acls but received %d", len(filters), len(resp.Results))
	}

	for _, result := range resp.Results {
		if err := kerr.ErrorForCode(result.ErrorCode); err != nil {
			errMsg := err.Error()
			if result.ErrorMessage != nil {
				errMsg = fmt.Sprintf("%s: %s", errMsg, *result.ErrorMessage)
			}
			return fmt.Errorf(errMsg)
		}
	}

	return nil
}
