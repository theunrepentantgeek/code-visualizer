package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// FilterBinaryFiles verifies that some files remain after the scan-time binary
// filter. Binary files are excluded during the filesystem scan when
// c.IncludeBinaryFiles is false; this stage exists only to surface a clear error
// when every file in the tree turned out to be binary.
//
// Returns *NoFilesAfterFilterError if c.Root contains no files.
func FilterBinaryFiles(c *CommonState) error {
	if c.IncludeBinaryFiles {
		return nil
	}

	if model.CountFiles(c.Root) == 0 {
		return &NoFilesAfterFilterError{Msg: NoFilesAfterFilterMsg}
	}

	return nil
}
