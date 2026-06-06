package treemap

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// FilterBinaryFiles strips binary files from c.Root unless the treemap state
// requests they be kept.
func FilterBinaryFiles(c *stages.CommonState, t *State) error {
	return eris.Wrap(stages.FilterBinaryFiles(c, t.IncludeBinaryFiles), "treemap: filter binary files")
}
