package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// CountAll returns the cumulative file and directory counts under root.
func CountAll(node *model.Directory) (files int, dirs int) {
	files = len(node.Files)
	for _, d := range node.Dirs {
		dirs++
		f, d2 := CountAll(d)
		files += f
		dirs += d2
	}

	return files, dirs
}

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

	count, _ := CountAll(c.Root)
	if count == 0 {
		return &NoFilesAfterFilterError{Msg: NoFilesAfterFilterMsg}
	}

	return nil
}
