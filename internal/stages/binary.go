package stages

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
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

// FilterBinaryFilesHelper removes binary files from the tree in place.
// Returns *NoFilesAfterFilterError if nothing remains.
func FilterBinaryFilesHelper(root *model.Directory) error {
	beforeCount, _ := CountAll(root)
	filtered := scan.FilterBinaryFiles(root)
	afterCount, _ := CountAll(filtered)
	excluded := beforeCount - afterCount
	slog.Debug("binary file filter", "excluded", excluded, "remaining", afterCount)

	if afterCount == 0 {
		return &NoFilesAfterFilterError{Msg: NoFilesAfterFilterMsg}
	}

	// Update root in place — avoid struct copy which would copy the mutex.
	root.Files = filtered.Files
	root.Dirs = filtered.Dirs

	return nil
}

// BinaryFilterToggler is implemented by per-viz state types that expose an
// "include binary files" flag. FilterBinaryFiles uses this to decide
// whether to run.
type BinaryFilterToggler interface {
	VizState
	IncludeBinary() bool
}

// FilterBinaryFiles is a pipeline.Stage that removes binary files from
// Common().Root unless the state's IncludeBinary() returns true.
func FilterBinaryFiles[S BinaryFilterToggler](s S) error {
	if s.IncludeBinary() {
		return nil
	}

	return FilterBinaryFilesHelper(s.Common().Root)
}

var _ pipeline.Stage[BinaryFilterToggler] = FilterBinaryFiles[BinaryFilterToggler]
