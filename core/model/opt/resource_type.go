// Package opt implements configuration options.
package opt

// ACLResourceTypeValidValues represents valid values for ACL resource type.
var ACLResourceTypeValidValues = []string{
	"any",
	"topic",
	"group",
	"cluster",
	"transactional_id",
	"delegation_token",
}
