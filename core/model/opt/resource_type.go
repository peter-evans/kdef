package opt

// Valid values for acl resource type
var ACLResourceTypeValidValues = []string{
	"any",
	"topic",
	"group",
	"cluster",
	"transactional_id",
	"delegation_token",
}
