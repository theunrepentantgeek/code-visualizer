package textlayout

import (
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

//nolint:gochecknoglobals // parsed once, read-only after init
var goFont = mustParseGoFont()

func mustParseGoFont() *truetype.Font {
	parsed, err := truetype.Parse(goregular.TTF)
	if err != nil {
		panic("textlayout: failed to parse Go Regular font: " + err.Error())
	}

	return parsed
}

// FontFace returns a Go Regular font face for the requested point size.
func FontFace(points float64) font.Face {
	return truetype.NewFace(goFont, &truetype.Options{Size: points})
}

// MeasureString returns the rendered width and line height for text at the given size.
func MeasureString(s string, points float64) (width, height float64) {
	face := FontFace(points)
	defer func() {
		if closer, ok := face.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}()

	drawer := &font.Drawer{Face: face}
	advance := drawer.MeasureString(s)

	return float64(advance >> 6), float64(face.Metrics().Height >> 6)
}
