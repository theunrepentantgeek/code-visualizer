// Package palette defines colour palettes for treemap visualisation
// and provides WCAG 2.0 luminance and contrast utilities.
package palette

import (
	"image/color"
	"math"
	"slices"
)

// PaletteName identifies a colour palette.
type PaletteName string

const (
	Categorization PaletteName = "categorization"
	Temperature    PaletteName = "temperature"
	GoodBad        PaletteName = "good-bad"
	Neutral        PaletteName = "neutral"
	Foliage        PaletteName = "foliage"
)

var validPalettes = map[PaletteName]struct{}{
	Categorization: {},
	Temperature:    {},
	GoodBad:        {},
	Neutral:        {},
	Foliage:        {},
}

func (p PaletteName) IsValid() bool {
	_, ok := validPalettes[p]

	return ok
}

// ColourPalette is the runtime representation of a palette.
type ColourPalette struct {
	Name        PaletteName
	Description string
	Colours     []color.RGBA
	Ordered     bool
}

// Neutral palette: 9 monochromatic steps from black to white.
var neutralPalette = ColourPalette{
	Name:        Neutral,
	Description: "Monochromatic greyscale (black → white). Good for quantity metrics with no directional meaning.",
	Ordered:     true,
	Colours: []color.RGBA{
		{R: 0, G: 0, B: 0, A: 255}, // black
		{R: 32, G: 32, B: 32, A: 255},
		{R: 64, G: 64, B: 64, A: 255},
		{R: 96, G: 96, B: 96, A: 255},
		{R: 128, G: 128, B: 128, A: 255}, // mid grey
		{R: 160, G: 160, B: 160, A: 255},
		{R: 192, G: 192, B: 192, A: 255},
		{R: 224, G: 224, B: 224, A: 255},
		{R: 255, G: 255, B: 255, A: 255}, // white
	},
}

var palettes = map[PaletteName]ColourPalette{
	Neutral:        neutralPalette,
	Categorization: categorizationPalette,
	Temperature:    temperaturePalette,
	GoodBad:        goodBadPalette,
	Foliage:        foliagePalette,
}

// Categorization palette: 12 visually distinct unordered colours (ColorBrewer Paired).
var categorizationPalette = ColourPalette{
	Name:        Categorization,
	Description: "12 visually distinct unordered colours (ColorBrewer Paired). Best for classification metrics.",
	Ordered:     false,
	Colours: []color.RGBA{
		{R: 166, G: 206, B: 227, A: 255},
		{R: 31, G: 120, B: 180, A: 255},
		{R: 178, G: 223, B: 138, A: 255},
		{R: 51, G: 160, B: 44, A: 255},
		{R: 251, G: 154, B: 153, A: 255},
		{R: 227, G: 26, B: 28, A: 255},
		{R: 253, G: 191, B: 111, A: 255},
		{R: 255, G: 127, B: 0, A: 255},
		{R: 202, G: 178, B: 214, A: 255},
		{R: 106, G: 61, B: 154, A: 255},
		{R: 255, G: 255, B: 153, A: 255},
		{R: 177, G: 89, B: 40, A: 255},
	},
}

// Temperature palette: 11 steps, dark blue → white → bright red (ColorBrewer RdBu diverging).
//
//nolint:dupl // palette declarations are structurally identical by design
var temperaturePalette = ColourPalette{
	Name:        Temperature,
	Description: "Diverging blue → white → red (ColorBrewer RdBu). Useful for bidirectional or time-based metrics.",
	Ordered:     true,
	Colours: []color.RGBA{
		{R: 5, G: 48, B: 97, A: 255},
		{R: 33, G: 102, B: 172, A: 255},
		{R: 67, G: 147, B: 195, A: 255},
		{R: 146, G: 197, B: 222, A: 255},
		{R: 209, G: 229, B: 240, A: 255},
		{R: 247, G: 247, B: 247, A: 255},
		{R: 253, G: 219, B: 199, A: 255},
		{R: 244, G: 165, B: 130, A: 255},
		{R: 214, G: 96, B: 77, A: 255},
		{R: 178, G: 24, B: 43, A: 255},
		{R: 103, G: 0, B: 31, A: 255},
	},
}

// Good/Bad palette: 13 steps, red → orange → yellow → green (ColorBrewer RdYlGn).
var goodBadPalette = ColourPalette{
	Name:        GoodBad,
	Description: "Sequential red → orange → yellow → green (ColorBrewer RdYlGn). Good for health or quality metrics.",
	Ordered:     true,
	Colours: []color.RGBA{
		{R: 165, G: 0, B: 38, A: 255},
		{R: 215, G: 48, B: 39, A: 255},
		{R: 244, G: 109, B: 67, A: 255},
		{R: 253, G: 174, B: 97, A: 255},
		{R: 254, G: 224, B: 139, A: 255},
		{R: 255, G: 255, B: 191, A: 255},
		{R: 255, G: 255, B: 255, A: 255},
		{R: 217, G: 239, B: 139, A: 255},
		{R: 166, G: 217, B: 106, A: 255},
		{R: 102, G: 189, B: 99, A: 255},
		{R: 26, G: 152, B: 80, A: 255},
		{R: 0, G: 104, B: 55, A: 255},
		{R: 0, G: 68, B: 27, A: 255},
	},
}

// Foliage palette: 11 steps, black → brown → orange → yellow → green (plant health).
//
//nolint:dupl // palette declarations are structurally identical by design
var foliagePalette = ColourPalette{
	Name:        Foliage,
	Description: "Sequential dead → brown → orange → yellow → green (plant health). Evokes code vitality.",
	Ordered:     true,
	Colours: []color.RGBA{
		{R: 15, G: 10, B: 5, A: 255},    // near black (dead)
		{R: 45, G: 25, B: 10, A: 255},   // very dark brown
		{R: 85, G: 45, B: 15, A: 255},   // dark brown
		{R: 130, G: 70, B: 20, A: 255},  // brown
		{R: 175, G: 95, B: 25, A: 255},  // dark orange
		{R: 210, G: 130, B: 30, A: 255}, // orange
		{R: 230, G: 175, B: 40, A: 255}, // yellow-orange
		{R: 240, G: 215, B: 50, A: 255}, // yellow
		{R: 165, G: 200, B: 50, A: 255}, // yellow-green
		{R: 80, G: 165, B: 40, A: 255},  // medium green
		{R: 25, G: 120, B: 20, A: 255},  // intense green
	},
}

// Names returns all registered palette names in sorted order.
func Names() []PaletteName {
	names := make([]PaletteName, 0, len(palettes))
	for n := range palettes {
		names = append(names, n)
	}

	slices.Sort(names)

	return names
}

// PaletteInfo holds the name and description of a palette.
type PaletteInfo struct {
	Name        PaletteName
	Description string
}

// Infos returns name and description for all registered palettes, sorted by name.
func Infos() []PaletteInfo {
	names := Names()
	infos := make([]PaletteInfo, 0, len(names))

	for _, n := range names {
		p := palettes[n]
		infos = append(infos, PaletteInfo{Name: n, Description: p.Description})
	}

	return infos
}

// GetPalette returns the ColourPalette for the given name.
// Returns a zero-value ColourPalette if the name is unknown.
func GetPalette(name PaletteName) ColourPalette {
	return palettes[name]
}

// RelativeLuminance computes the WCAG 2.0 relative luminance of a colour.
func RelativeLuminance(c color.RGBA) float64 {
	r := linearize(float64(c.R) / 255.0)
	g := linearize(float64(c.G) / 255.0)
	b := linearize(float64(c.B) / 255.0)

	return 0.2126*r + 0.7152*g + 0.0722*b
}

// ContrastRatio returns the WCAG 2.0 contrast ratio between two colours.
func ContrastRatio(a, b color.RGBA) float64 {
	l1 := RelativeLuminance(a)
	l2 := RelativeLuminance(b)
	lighter := math.Max(l1, l2)
	darker := math.Min(l1, l2)

	return (lighter + 0.05) / (darker + 0.05)
}

func linearize(v float64) float64 {
	if v <= 0.03928 {
		return v / 12.92
	}

	return math.Pow((v+0.055)/1.055, 2.4)
}
