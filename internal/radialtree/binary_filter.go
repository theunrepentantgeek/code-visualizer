package radialtree

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// FilterBinaryFiles strips binary files from c.Root unless the radialtree
// state requests they be kept.
func FilterBinaryFiles(c *stages.CommonState, r *State) error {
	return eris.Wrap(stages.FilterBinaryFiles(c, r.IncludeBinaryFiles), "radialtree: filter binary files")
}
