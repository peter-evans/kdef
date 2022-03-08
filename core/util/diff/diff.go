// Package diff implements functions to compute a line-oriented diff.
package diff

import (
	"strings"

	"github.com/peter-evans/patience"
)

// LineOriented computes the line-oriented diff from source to destination strings.
func LineOriented(src string, dst string) string {
	diff := patience.Diff(
		strings.Split(src, "\n"),
		strings.Split(dst, "\n"),
	)
	return patience.DiffText(diff)
}
