package opt

type Assignments int8

const (
	UnsupportedAssignments Assignments = 0
	NoAssignments          Assignments = 1
	BrokerAssignments      Assignments = 2
	RackAssignments        Assignments = 3
)

// Valid values for assignments
var AssignmentsValidValues = []string{"none", "broker", "rack"}

// Parse an assignments option from a string
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
