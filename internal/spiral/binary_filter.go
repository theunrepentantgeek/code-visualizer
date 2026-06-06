package spiral

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// FilterBinaryFiles strips binary files from c.Root unless the spiral state
// requests they be kept.
func FilterBinaryFiles(c *stages.CommonState, p *State) error {
	return eris.Wrap(stages.FilterBinaryFiles(c, p.IncludeBinaryFiles), "spiral: filter binary files")
}
