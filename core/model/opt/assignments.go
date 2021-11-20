// Package opt implements configuration options.
package opt

// Assignments represents the format in which assignments are exported.
type Assignments int8

// Assignments types.
const (
	UnsupportedAssignments Assignments = 0
	NoAssignments          Assignments = 1
	BrokerAssignments      Assignments = 2
	RackAssignments        Assignments = 3
)

// AssignmentsValidValues represents valid values for assignments.
var AssignmentsValidValues = []string{"none", "broker", "rack"}

// ParseAssignments parses an assignments option from a string.
func ParseAssignments(assignments string) Assignments {
	switch assignments {
	case "none":
		return NoAssignments
	case "broker":
		return BrokerAssignments
	case "rack":
		return RackAssignments
	default:
		return UnsupportedAssignments
	}
}
