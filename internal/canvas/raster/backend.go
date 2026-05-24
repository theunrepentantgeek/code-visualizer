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

	"github.com/fogleman/gg"
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"
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
	pos model.Position, size model.Size, fill, border model.Fill, borderWidth float64,
) {
	switch f := fill.(type) {
	case model.SolidFill:
		r.dc.SetColor(nrgba(f.Color))
		r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
		r.dc.Fill()
	case model.RadialGradientFill:
		r.drawRadialGradientRect(pos, size, f)
	default:
		r.dc.SetColor(nrgba(color.RGBA{A: 255}))
		r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
		r.dc.Fill()
	}

	if borderWidth > 0 {
		borderColour := solidColor(border)
		r.dc.SetColor(nrgba(borderColour))
		r.dc.SetLineWidth(borderWidth)
		r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
		r.dc.Stroke()
	}
}

func (r *rasterBackend) drawRadialGradientRect(
	pos model.Position, size model.Size, grad model.RadialGradientFill,
) {
	fx := pos.X + grad.Focus.X*size.Width
	fy := pos.Y + grad.Focus.Y*size.Height
	maxDist := maxCornerDist(fx, fy, pos.X, pos.Y, size.Width, size.Height)

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

	invMax := 1.0 / maxDist

	for py := y0; py < y1; py++ {
		dy := float64(py) + 0.5 - fy

		for px := x0; px < x1; px++ {
			dx := float64(px) + 0.5 - fx
			dist := math.Sqrt(dx*dx + dy*dy)
			t := min(dist*invMax, 1.0)
			img.SetRGBA(px, py, lerpColour(grad.Center, grad.Edge, t))
		}
	}
}

func maxCornerDist(fx, fy, rx, ry, w, h float64) float64 {
	corners := [4][2]float64{{rx, ry}, {rx + w, ry}, {rx, ry + h}, {rx + w, ry + h}}
	maxDist := 0.0

	for _, corner := range corners {
		dx := corner[0] - fx
		dy := corner[1] - fy
		dist := math.Sqrt(dx*dx + dy*dy)

		if dist > maxDist {
			maxDist = dist
		}
	}

	return maxDist
}

func lerpColour(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: uint8(float64(a.A) + (float64(b.A)-float64(a.A))*t),
	}
}

func (r *rasterBackend) DrawDisc(
	center model.Position, radius float64, fill, border model.Fill, borderWidth float64,
) {
	fillColour := solidColor(fill)
	r.dc.SetColor(nrgba(fillColour))
	r.dc.DrawCircle(center.X, center.Y, radius)
	r.dc.Fill()

	if borderWidth > 0 {
		borderColour := solidColor(border)
		r.dc.SetColor(nrgba(borderColour))
		r.dc.SetLineWidth(borderWidth)
		r.dc.DrawCircle(center.X, center.Y, radius)
		r.dc.Stroke()
	}
}

// solidColor extracts the colour from a Fill, falling back to opaque black.
func solidColor(f model.Fill) color.RGBA {
	switch v := f.(type) {
	case model.SolidFill:
		return v.Color
	case model.RadialGradientFill:
		return v.Center
	default:
		return color.RGBA{A: 255}
	}
}

func (r *rasterBackend) DrawLine(from, to model.Position, stroke color.RGBA, strokeWidth float64) {
	r.dc.SetColor(nrgba(stroke))
	r.dc.SetLineWidth(strokeWidth)
	r.dc.DrawLine(from.X, from.Y, to.X, to.Y)
	r.dc.Stroke()
}

func (r *rasterBackend) DrawPath(points []model.Position, stroke color.RGBA, strokeWidth float64) {
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
	pos model.Position,
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
	center model.Position,
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

	totalAngle := float64(len([]rune(text))) * fontSize * 0.6 / arcRadius
	startAngle := -math.Pi/2.0 - totalAngle/2.0
	charAngle := totalAngle / float64(len([]rune(text)))

	for i, ch := range text {
		angle := startAngle + float64(i)*charAngle + charAngle/2.0
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
