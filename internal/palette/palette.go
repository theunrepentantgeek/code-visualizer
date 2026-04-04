package palette

import "image/color"

// PaletteName identifies a colour palette.
type PaletteName string

const (
	Categorization PaletteName = "categorization"
	Temperature    PaletteName = "temperature"
	GoodBad        PaletteName = "good-bad"
	Neutral        PaletteName = "neutral"
)

var validPalettes = map[PaletteName]struct{}{
	Categorization: {},
	Temperature:    {},
	GoodBad:        {},
	Neutral:        {},
}

func (p PaletteName) IsValid() bool {
	_, ok := validPalettes[p]
	return ok
}

// ColourPalette is the runtime representation of a palette.
type ColourPalette struct {
	Name    PaletteName
	Colours []color.RGBA
	Ordered bool
}
