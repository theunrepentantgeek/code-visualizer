package scatter

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// FilterBinaryFiles strips binary files from c.Root unless the scatter state
// requests they be kept.
func FilterBinaryFiles(c *stages.CommonState, x *State) error {
	return stages.FilterBinaryFiles(c, x.IncludeBinaryFiles)
}
