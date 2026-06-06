package treemap

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// FilterBinaryFiles strips binary files from c.Root unless the treemap state
// requests they be kept.
func FilterBinaryFiles(c *stages.CommonState, t *State) error {
	return stages.FilterBinaryFiles(c, t.IncludeBinaryFiles)
}
