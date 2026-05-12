package legendlayout

import (
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// StringMeasurer measures the rendered width and height of a string.
// Implementations may use actual font metrics or a fixed-width approximation.
type StringMeasurer interface {
	MeasureString(s string) (w, h float64)
}

// basicFontMeasurer uses the 7×13 bitmap font from golang.org/x/image,
// which is the same default font that gg.NewContext uses.
type basicFontMeasurer struct {
	face font.Face
}

// NewBasicMeasurer returns a StringMeasurer backed by the standard 7×13
// bitmap font. This matches the default font used by gg.NewContext(1, 1),
// so measurements are identical to the previous gg-based implementation.
func NewBasicMeasurer() StringMeasurer {
	return &basicFontMeasurer{face: basicfont.Face7x13}
}

func (m *basicFontMeasurer) MeasureString(s string) (w, h float64) {
	d := &font.Drawer{Face: m.face}
	advance := d.MeasureString(s)

	return float64(advance >> 6), float64(m.face.Metrics().Height >> 6)
}
