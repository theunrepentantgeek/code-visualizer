// Package raster implements the model.Backend interface for raster
// output formats (PNG, JPG) using the fogleman/gg graphics library.
package raster

import (
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/fogleman/gg"
	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/canvas/model"
)

const jpegQuality = 95

type rasterBackend struct {
	dc *gg.Context
}

// New creates a raster backend with the given dimensions.
func New(width, height int) model.Backend {
	dc := gg.NewContext(width, height)

	return &rasterBackend{dc: dc}
}

func (r *rasterBackend) DrawRectangle(
	pos model.Position, size model.Size, fill, border color.RGBA, borderWidth float64,
) {
	r.dc.SetColor(fill)
	r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
	r.dc.Fill()

	if borderWidth > 0 {
		r.dc.SetColor(border)
		r.dc.SetLineWidth(borderWidth)
		r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
		r.dc.Stroke()
	}
}

func (r *rasterBackend) DrawDisc(center model.Position, radius float64, fill, border color.RGBA, borderWidth float64) {
	r.dc.SetColor(fill)
	r.dc.DrawCircle(center.X, center.Y, radius)
	r.dc.Fill()

	if borderWidth > 0 {
		r.dc.SetColor(border)
		r.dc.SetLineWidth(borderWidth)
		r.dc.DrawCircle(center.X, center.Y, radius)
		r.dc.Stroke()
	}
}

func (r *rasterBackend) DrawLine(from, to model.Position, stroke color.RGBA, strokeWidth float64) {
	r.dc.SetColor(stroke)
	r.dc.SetLineWidth(strokeWidth)
	r.dc.DrawLine(from.X, from.Y, to.X, to.Y)
	r.dc.Stroke()
}

func (r *rasterBackend) DrawPath(points []model.Position, stroke color.RGBA, strokeWidth float64) {
	if len(points) < 2 {
		return
	}

	r.dc.SetColor(stroke)
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
	r.dc.SetColor(ink)
	// fontSize is accepted for interface conformance but requires a loaded
	// font face to take effect; the default gg font ignores size changes.
	_ = fontSize

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

	r.dc.SetColor(ink)

	arcRadius := radius - 14.0
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
		r.dc.DrawStringAnchored(string(ch), cx, cy, 0.5, 0.5)
		r.dc.Pop()
	}
}

func (r *rasterBackend) DrawLegend(data model.LegendData, canvasW, canvasH int) {
	drawLegend(r.dc, data, canvasW, canvasH)
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
