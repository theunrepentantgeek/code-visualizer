package render

import (
	"math"
	"unicode/utf8"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"
)

const (
	// bubbleMinFontSize is the smallest readable font size in pixels.
	// Labels that would require a size below this are hidden.
	bubbleMinFontSize = 7.0

	// bubbleDefaultFontSize is the starting font size before arc constraints
	// clamp it down.
	bubbleDefaultFontSize = 14.0

	// bubbleMaxArcFraction is the maximum fraction of the full circle
	// that a label may span (90° = π/2).
	bubbleMaxArcFraction = math.Pi / 2.0
)

// glyphPos describes a single character positioned along an arc.
type glyphPos struct {
	Char  rune
	Angle float64 // radians from top-centre (-π/2 base)
	X, Y  float64 // relative to circle centre
}

// parsedFont is the TrueType font parsed once at init time.
var parsedFont *truetype.Font

func init() {
	var err error

	parsedFont, err = truetype.Parse(goregular.TTF)
	if err != nil {
		panic("render: failed to parse goregular font: " + err.Error())
	}
}

// loadBubbleFontFace returns a font.Face for goregular at the given pixel size.
func loadBubbleFontFace(size float64) font.Face {
	return truetype.NewFace(parsedFont, &truetype.Options{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

// computeArcFontSize computes the font size that makes label fit within
// bubbleMaxArcFraction of a circle with the given radius.
// Returns 0 if the label cannot be rendered at bubbleMinFontSize.
func computeArcFontSize(label string, radius float64) float64 {
	if utf8.RuneCountInString(label) == 0 {
		return 0
	}

	arcR := radius - bubbleLabelInset
	if arcR <= 0 {
		return 0
	}

	maxArcLen := arcR * bubbleMaxArcFraction
	size := bubbleDefaultFontSize

	return clampFontToArc(label, size, maxArcLen)
}

// clampFontToArc iteratively reduces font size until the label fits within
// maxArcLen, returning 0 if it cannot reach bubbleMinFontSize.
func clampFontToArc(label string, size, maxArcLen float64) float64 {
	for size >= bubbleMinFontSize {
		face := loadBubbleFontFace(size)
		tw := measureStringWidth(label, face)

		if tw <= maxArcLen {
			return size
		}

		// Scale down proportionally with a small margin.
		size = size * maxArcLen / tw * 0.95

		if size < bubbleMinFontSize {
			return 0
		}
	}

	return 0
}

// measureStringWidth returns the total advance width of a string using
// per-glyph advances and kerning.
func measureStringWidth(s string, face font.Face) float64 {
	var width fixedAccum

	prev := rune(-1)

	for _, r := range s {
		if prev >= 0 {
			width.add(face.Kern(prev, r))
		}

		adv, ok := face.GlyphAdvance(r)
		if ok {
			width.add(adv)
		}

		prev = r
	}

	return width.total()
}

// fixedAccum accumulates fixed-point Int26_6 advances into a float64 sum.
type fixedAccum struct {
	sum fixed.Int26_6
}

func (f *fixedAccum) add(v fixed.Int26_6) {
	f.sum += v
}

func (f *fixedAccum) total() float64 {
	return float64(f.sum) / 64.0
}

// computeGlyphPositions returns arc-positioned glyphs centred at the top of a
// circle. Each glyph has an angle and X/Y offset relative to the circle centre.
func computeGlyphPositions(label string, face font.Face, arcRadius float64) []glyphPos {
	advances := collectAdvances(label, face)
	totalWidth := sumAdvances(advances)

	if arcRadius <= 0 || totalWidth <= 0 {
		return nil
	}

	// Total angular span of the text.
	totalAngle := totalWidth / arcRadius

	// Start angle: centred at top of circle (-π/2).
	startAngle := -math.Pi/2.0 - totalAngle/2.0

	return placeGlyphs(label, advances, startAngle, arcRadius)
}

// charAdvance pairs a rune with its advance width in pixels.
type charAdvance struct {
	r       rune
	advance float64
}

// collectAdvances measures per-character advances with kerning.
func collectAdvances(label string, face font.Face) []charAdvance {
	runes := []rune(label)
	result := make([]charAdvance, len(runes))

	for i, r := range runes {
		adv, _ := face.GlyphAdvance(r)
		charWidth := float64(adv) / 64.0

		if i > 0 {
			kern := face.Kern(runes[i-1], r)
			charWidth += float64(kern) / 64.0
		}

		result[i] = charAdvance{r: r, advance: charWidth}
	}

	return result
}

// sumAdvances totals the advance widths.
func sumAdvances(advances []charAdvance) float64 {
	var total float64
	for _, a := range advances {
		total += a.advance
	}

	return total
}

// placeGlyphs positions each glyph along the arc, returning a glyphPos slice.
func placeGlyphs(label string, advances []charAdvance, startAngle, arcRadius float64) []glyphPos {
	_ = label // used only for documentation; runes come from advances

	positions := make([]glyphPos, len(advances))
	currentAngle := startAngle

	for i, a := range advances {
		// Place glyph at centre of its angular span.
		glyphAngle := a.advance / arcRadius
		midAngle := currentAngle + glyphAngle/2.0

		positions[i] = glyphPos{
			Char:  a.r,
			Angle: midAngle,
			X:     arcRadius * math.Cos(midAngle),
			Y:     arcRadius * math.Sin(midAngle),
		}

		currentAngle += glyphAngle
	}

	return positions
}
