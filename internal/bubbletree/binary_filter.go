package bubbletree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// FilterBinaryFiles strips binary files from c.Root unless the bubbletree state
// requests they be kept.
func FilterBinaryFiles(c *stages.CommonState, b *State) error {
	return stages.FilterBinaryFiles(c, b.IncludeBinaryFiles)
}
