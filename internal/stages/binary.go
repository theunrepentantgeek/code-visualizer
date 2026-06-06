package stages

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
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

// FilterBinaryFiles removes binary files from c.Root in place unless
// c.IncludeBinaryFiles is true. Returns *NoFilesAfterFilterError if nothing
// remains.
func FilterBinaryFiles(c *CommonState) error {
	if c.IncludeBinaryFiles {
		return nil
	}

	beforeCount, _ := CountAll(c.Root)
	filtered := scan.FilterBinaryFiles(c.Root)
	afterCount, _ := CountAll(filtered)
	excluded := beforeCount - afterCount
	slog.Debug("binary file filter", "excluded", excluded, "remaining", afterCount)

	if afterCount == 0 {
		return &NoFilesAfterFilterError{Msg: NoFilesAfterFilterMsg}
	}

	// Update root in place — avoid struct copy which would copy the mutex.
	c.Root.Files = filtered.Files
	c.Root.Dirs = filtered.Dirs

	return nil
}
