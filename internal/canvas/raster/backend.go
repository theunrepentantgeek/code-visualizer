// Package raster implements the model.Backend interface for raster
// output formats (PNG, JPG) using the fogleman/gg graphics library.
package raster

import (
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/fogleman/gg"
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

const jpegQuality = 95

// defaultFontSize is the font size used when callers pass fontSize <= 0,
// indicating "use the backend's default".
const defaultFontSize = 12.0

type rasterBackend struct {
	dc *gg.Context
}

// New creates a raster backend with the given dimensions.
func New(width, height int) model.Backend {
	dc := gg.NewContext(width, height)

	return &rasterBackend{dc: dc}
}

func (r *rasterBackend) DrawRectangle(
	pos geometry.Point, size model.Size, fill, border model.Fill, borderWidth float64,
) {
	if f, ok := fill.(model.RadialGradientFill); ok {
		r.drawRadialGradientRect(pos, size, f)
	} else {
		r.dc.SetColor(nrgba(model.SolidColor(fill)))
		r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
		r.dc.Fill()
	}

	if borderWidth > 0 {
		borderColour := model.SolidColor(border)
		r.dc.SetColor(nrgba(borderColour))
		r.dc.SetLineWidth(borderWidth)
		r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
		r.dc.Stroke()
	}
}

func (r *rasterBackend) drawRadialGradientRect(
	pos geometry.Point, size model.Size, grad model.RadialGradientFill,
) {
	focus := geometry.Point{
		X: pos.X + grad.Focus.X*size.Width,
		Y: pos.Y + grad.Focus.Y*size.Height,
	}
	maxDist := maxCornerDist(focus.X, focus.Y, pos.X, pos.Y, size.Width, size.Height)

	if maxDist == 0 {
		r.dc.SetColor(nrgba(grad.Center))
		r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
		r.dc.Fill()

		return
	}

	// Render gradient pixel-by-pixel to avoid gg's broken Push/Clip/Pop.
	img, ok := r.dc.Image().(*image.RGBA)
	if !ok {
		return
	}

	x0 := int(pos.X)
	y0 := int(pos.Y)
	x1 := int(pos.X + size.Width)
	y1 := int(pos.Y + size.Height)
	bounds := img.Bounds()
	x0 = max(x0, bounds.Min.X)
	y0 = max(y0, bounds.Min.Y)
	x1 = min(x1, bounds.Max.X)
	y1 = min(y1, bounds.Max.Y)

	lerp := newGradientLerp(grad.Center, grad.Edge)
	renderRadialGradientPixels(
		img, image.Rect(x0, y0, x1, y1), focus.X, focus.Y, 1.0/maxDist, lerp, radialClip{},
	)
}

// maxCornerDist returns the maximum distance from point (fx,fy) to any corner
// of the rectangle with top-left (rx,ry), width w, and height h.
//
// The maximum of dx²+dy² over the four corners decomposes as
// max(dx0²,dx1²) + max(dy0²,dy1²) because dx and dy are independent, so only
// one math.Sqrt is required instead of four.
func maxCornerDist(fx, fy, rx, ry, w, h float64) float64 {
	dx0 := rx - fx
	dx1 := rx + w - fx
	dy0 := ry - fy
	dy1 := ry + h - fy

	return math.Sqrt(max(dx0*dx0, dx1*dx1) + max(dy0*dy0, dy1*dy1))
}

func (r *rasterBackend) DrawDisc(
	center geometry.Point, radius float64, fill, border model.Fill, borderWidth float64,
) {
	if f, ok := fill.(model.RadialGradientFill); ok {
		r.drawRadialGradientDisc(center, radius, f)
	} else {
		r.dc.SetColor(nrgba(model.SolidColor(fill)))
		r.dc.DrawCircle(center.X, center.Y, radius)
		r.dc.Fill()
	}

	if borderWidth > 0 {
		borderColour := model.SolidColor(border)
		r.dc.SetColor(nrgba(borderColour))
		r.dc.SetLineWidth(borderWidth)
		r.dc.DrawCircle(center.X, center.Y, radius)
		r.dc.Stroke()
	}
}

func (r *rasterBackend) DrawPolygon(
	points []geometry.Point, fill, border model.Fill, borderWidth float64,
) {
	if len(points) < 3 {
		return
	}

	r.drawPolygonPath(points)

	if f, ok := fill.(model.RadialGradientFill); ok && r.drawRadialGradientPolygon(points, f) {
		if borderWidth <= 0 {
			r.dc.ClearPath()
		}
	} else {
		r.dc.SetColor(nrgba(model.SolidColor(fill)))

		if borderWidth > 0 {
			r.dc.FillPreserve()
		} else {
			r.dc.Fill()
		}
	}

	if borderWidth > 0 {
		r.dc.SetColor(nrgba(model.SolidColor(border)))
		r.dc.SetLineWidth(borderWidth)
		r.dc.Stroke()
	}
}

func (r *rasterBackend) DrawFilledPath(loops [][]geometry.Point, fill color.RGBA) {
	r.dc.SetFillRuleEvenOdd()
	defer r.dc.SetFillRuleWinding()

	for _, loop := range loops {
		if len(loop) < 3 {
			continue
		}

		r.drawPolygonPath(loop)
	}

	r.dc.SetColor(nrgba(fill))
	r.dc.Fill()
}

func (r *rasterBackend) drawPolygonPath(points []geometry.Point) {
	r.dc.MoveTo(points[0].X, points[0].Y)

	for _, point := range points[1:] {
		r.dc.LineTo(point.X, point.Y)
	}

	r.dc.ClosePath()
}

func (r *rasterBackend) drawRadialGradientPolygon(
	points []geometry.Point, grad model.RadialGradientFill,
) bool {
	img, ok := r.dc.Image().(*image.RGBA)
	if !ok {
		return false
	}

	minX, maxX := points[0].X, points[0].X

	minY, maxY := points[0].Y, points[0].Y
	for _, point := range points[1:] {
		minX = min(minX, point.X)
		maxX = max(maxX, point.X)
		minY = min(minY, point.Y)
		maxY = max(maxY, point.Y)
	}

	// Focus is relative to the polygon's bounding box; the farthest vertex
	// establishes the radius, matching rectangle gradient normalization.
	focus := geometry.Point{
		X: minX + grad.Focus.X*(maxX-minX),
		Y: minY + grad.Focus.Y*(maxY-minY),
	}

	maxDist := 0.0
	for _, point := range points {
		maxDist = max(maxDist, math.Hypot(point.X-focus.X, point.Y-focus.Y))
	}

	if maxDist == 0 {
		return true
	}

	bounds := img.Bounds()
	x0 := max(int(math.Floor(minX)), bounds.Min.X)
	y0 := max(int(math.Floor(minY)), bounds.Min.Y)
	x1 := min(int(math.Ceil(maxX)), bounds.Max.X)
	y1 := min(int(math.Ceil(maxY)), bounds.Max.Y)

	lerp := newGradientLerp(grad.Center, grad.Edge)
	renderPolygonGradientPixels(
		img, image.Rect(x0, y0, x1, y1), points, focus.X, focus.Y, 1.0/maxDist, lerp,
	)

	return true
}

func (r *rasterBackend) drawRadialGradientDisc(
	center geometry.Point, radius float64, grad model.RadialGradientFill,
) {
	if radius == 0 {
		return
	}

	img, ok := r.dc.Image().(*image.RGBA)
	if !ok {
		r.dc.SetColor(nrgba(grad.Center))
		r.dc.DrawCircle(center.X, center.Y, radius)
		r.dc.Fill()

		return
	}

	focus := geometry.Point{
		X: center.X + (grad.Focus.X-0.5)*2*radius,
		Y: center.Y + (grad.Focus.Y-0.5)*2*radius,
	}

	bounds := img.Bounds()
	x0 := max(int(center.X-radius), bounds.Min.X)
	y0 := max(int(center.Y-radius), bounds.Min.Y)
	x1 := min(int(center.X+radius)+1, bounds.Max.X)
	y1 := min(int(center.Y+radius)+1, bounds.Max.Y)

	lerp := newGradientLerp(grad.Center, grad.Edge)
	renderRadialGradientPixels(
		img, image.Rect(x0, y0, x1, y1),
		focus.X, focus.Y, 1.0/radius, lerp,
		radialClip{cx: center.X, cy: center.Y, r: radius},
	)
}

func (r *rasterBackend) DrawLine(from, to geometry.Point, stroke color.RGBA, strokeWidth float64) {
	r.dc.SetColor(nrgba(stroke))
	r.dc.SetLineWidth(strokeWidth)
	r.dc.DrawLine(from.X, from.Y, to.X, to.Y)
	r.dc.Stroke()
}

func (r *rasterBackend) DrawPath(points []geometry.Point, stroke color.RGBA, strokeWidth float64) {
	if len(points) < 2 {
		return
	}

	r.dc.SetColor(nrgba(stroke))
	r.dc.SetLineWidth(strokeWidth)
	r.dc.MoveTo(points[0].X, points[0].Y)

	for _, p := range points[1:] {
		r.dc.LineTo(p.X, p.Y)
	}

	r.dc.Stroke()
}

func (r *rasterBackend) DrawText(
	pos geometry.Point,
	text string,
	ink color.RGBA,
	fontSize float64,
	anchor model.TextAnchor,
	rotation float64,
) {
	if fontSize <= 0 {
		fontSize = defaultFontSize
	}

	face := textlayout.FontFace(fontSize)
	if closer, ok := face.(interface{ Close() error }); ok {
		defer func() { _ = closer.Close() }()
	}

	r.dc.SetFontFace(face)
	r.dc.SetColor(nrgba(ink))

	ax := anchorX(anchor)

	if rotation != 0 {
		r.dc.RotateAbout(rotation, pos.X, pos.Y)
	}

	r.dc.DrawStringAnchored(text, pos.X, pos.Y, ax, 0.5)

	if rotation != 0 {
		r.dc.RotateAbout(-rotation, pos.X, pos.Y)
	}
}

func (r *rasterBackend) DrawArcText(
	center geometry.Point,
	radius float64,
	text string,
	ink color.RGBA,
	fontSize float64,
) {
	if text == "" || radius <= 0 {
		return
	}

	if fontSize <= 0 {
		fontSize = defaultFontSize
	}

	r.dc.SetFontFace(textlayout.FontFace(fontSize))
	r.dc.SetColor(nrgba(ink))

	arcRadius := radius - model.ArcTextInset
	if arcRadius <= 0 {
		return
	}

	forEachArcTextRune(text, fontSize, arcRadius, func(ch rune, angle float64) {
		cx := center.X + arcRadius*math.Cos(angle)
		cy := center.Y + arcRadius*math.Sin(angle)

		r.dc.Push()
		r.dc.RotateAbout(angle+math.Pi/2.0, cx, cy)
		// gg's DrawStringAnchored places the baseline at cy + ay*h. Using
		// ay=0.5 puts the baseline at the rim of the underlying circle so
		// non-descender letters touch the rim. Use ay=0.25 to match the
		// SVG backend's dominant-baseline="middle" behaviour, which lifts
		// the baseline so descenders just graze the rim instead.
		r.dc.DrawStringAnchored(string(ch), cx, cy, 0.5, 0.25)
		r.dc.Pop()
	})
}

func forEachArcTextRune(text string, fontSize, arcRadius float64, yield func(ch rune, angle float64)) {
	if text == "" {
		return
	}

	// Use a rune count rather than []rune(text) to avoid a heap allocation,
	// and track the rune index separately so character placement is correct
	// for multi-byte (non-ASCII) code points.
	n := utf8.RuneCountInString(text)
	totalAngle := float64(n) * fontSize * 0.6 / arcRadius
	startAngle := -math.Pi/2.0 - totalAngle/2.0
	charAngle := totalAngle / float64(n)

	var runeIndex int
	for _, ch := range text {
		yield(ch, startAngle+float64(runeIndex)*charAngle+charAngle/2.0)
		runeIndex++
	}
}

func (r *rasterBackend) Finish(outputPath string) error {
	ext := strings.ToLower(filepath.Ext(outputPath))

	switch ext {
	case ".png":
		return eris.Wrap(r.dc.SavePNG(outputPath), "failed to save PNG")
	case ".jpg", ".jpeg":
		return r.saveJPG(outputPath)
	default:
		return eris.Errorf("unsupported raster format %q", ext)
	}
}

func (r *rasterBackend) saveJPG(path string) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return eris.Wrap(err, "failed to create JPEG file")
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = eris.Wrap(closeErr, "failed to close JPEG file")
		}
	}()

	if err := jpeg.Encode(f, r.dc.Image(), &jpeg.Options{Quality: jpegQuality}); err != nil {
		return eris.Wrap(err, "failed to encode JPEG")
	}

	return nil
}

func anchorX(a model.TextAnchor) float64 {
	switch a {
	case model.AnchorMiddle:
		return 0.5
	case model.AnchorEnd:
		return 1.0
	default:
		return 0.0
	}
}

// nrgba converts a color.RGBA value — stored as non-premultiplied throughout
// this project — to color.NRGBA so that gg's raster painter receives correctly
// premultiplied values when it calls RGBA() internally.
// Without this conversion, semi-transparent colours produce incorrect results
// because color.RGBA.RGBA() treats R,G,B as already premultiplied, but in this
// codebase they are the actual (non-premultiplied) channel values.
func nrgba(c color.RGBA) color.NRGBA {
	return color.NRGBA(c)
}
