package render

import (
	"github.com/fogleman/gg"
	"github.com/rotisserie/eris"
)

// saveContextPNG saves the gg context as a PNG file.
func saveContextPNG(dc *gg.Context, path string) error {
	return eris.Wrap(dc.SavePNG(path), "failed to save PNG")
}
