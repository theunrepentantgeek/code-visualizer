package spiral

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// FilterBinaryFiles strips binary files from c.Root unless the spiral state
// requests they be kept.
func FilterBinaryFiles(c *stages.CommonState, p *State) error {
	return stages.FilterBinaryFiles(c, p.IncludeBinaryFiles)
}
