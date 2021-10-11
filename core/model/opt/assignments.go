package opt

type Assignments int8

const (
	UnsupportedAssignments Assignments = -1
	NoAssignments          Assignments = 0
	BrokerAssignments      Assignments = 1
	RackAssignments        Assignments = 2
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
