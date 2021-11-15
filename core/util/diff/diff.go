// Package diff implements functions to compute a line-oriented diff.
package diff

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// LineOriented computes the line-oriented diff from source to destination strings.
func LineOriented(src string, dst string) string {
	dmp := diffmatchpatch.New()
	// Default timeout is one second.
	dmp.DiffTimeout = time.Second * 2

	srcRunes, dstRunes, lineArray := dmp.DiffLinesToRunes(src, dst)
	diffs := dmp.DiffMainRunes(srcRunes, dstRunes, false)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	return formatDiff(diffs)
}

func formatDiff(diffs []diffmatchpatch.Diff) string {
	var buf bytes.Buffer
	for _, d := range diffs {
		lines := strings.Split(d.Text, "\n")
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			switch d.Type {
			case diffmatchpatch.DiffEqual:
				buf.WriteString(" ")
			case diffmatchpatch.DiffDelete:
				buf.WriteString("-")
			case diffmatchpatch.DiffInsert:
				buf.WriteString("+")
			}
			buf.WriteString(fmt.Sprintf("%s\n", line))
		}
	}
	return buf.String()
}
