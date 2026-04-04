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

// Neutral palette: 9 monochromatic steps from black to white.
var neutralPalette = ColourPalette{
	Name:    Neutral,
	Ordered: true,
	Colours: []color.RGBA{
		{R: 0, G: 0, B: 0, A: 255},         // black
		{R: 32, G: 32, B: 32, A: 255},
		{R: 64, G: 64, B: 64, A: 255},
		{R: 96, G: 96, B: 96, A: 255},
		{R: 128, G: 128, B: 128, A: 255},    // mid grey
		{R: 160, G: 160, B: 160, A: 255},
		{R: 192, G: 192, B: 192, A: 255},
		{R: 224, G: 224, B: 224, A: 255},
		{R: 255, G: 255, B: 255, A: 255},    // white
	},
}

var palettes = map[PaletteName]ColourPalette{
	Neutral: neutralPalette,
}

// GetPalette returns the ColourPalette for the given name.
// Returns a zero-value ColourPalette if the name is unknown.
func GetPalette(name PaletteName) ColourPalette {
	return palettes[name]
}
