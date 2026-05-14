// Package svg implements the model.Backend interface for SVG vector output
// using direct XML generation.
package svg

import (
	"bytes"
	"fmt"
	"html"
	"image/color"
	"math"
	"os"
	"strings"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

// defaultFontSize is the font size used when callers pass fontSize <= 0,
// indicating "use the backend's default".
const defaultFontSize = 12.0

type svgBackend struct {
	width  int
	height int
	buf    bytes.Buffer
}

// New creates an SVG backend with the given dimensions.
func New(width, height int) model.Backend {
	b := &svgBackend{width: width, height: height}
	b.writeHeader()

	return b
}

func (s *svgBackend) writeHeader() {
	fmt.Fprintf(
		&s.buf,
		`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`+"\n",
		s.width, s.height,
	)
}

func (s *svgBackend) DrawRectangle(
	pos model.Position, size model.Size, fill, border color.RGBA, borderWidth float64,
) {
	fmt.Fprintf(
		&s.buf,
		`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s" stroke="%s" stroke-width="%.1f"/>`+"\n",
		pos.X, pos.Y, size.Width, size.Height,
		rgbaToCSS(fill), rgbaToCSS(border), borderWidth,
	)
}

func (s *svgBackend) DrawDisc(center model.Position, radius float64, fill, border color.RGBA, borderWidth float64) {
	fmt.Fprintf(
		&s.buf,
		`<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s" stroke="%s" stroke-width="%.1f"/>`+"\n",
		center.X, center.Y, radius,
		rgbaToCSS(fill), rgbaToCSS(border), borderWidth,
	)
}

func (s *svgBackend) DrawLine(from, to model.Position, stroke color.RGBA, strokeWidth float64) {
	fmt.Fprintf(
		&s.buf,
		`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="%.1f"/>`+"\n",
		from.X, from.Y, to.X, to.Y,
		rgbaToCSS(stroke), strokeWidth,
	)
}

func (s *svgBackend) DrawPath(points []model.Position, stroke color.RGBA, strokeWidth float64) {
	if len(points) < 2 {
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, "M %.1f %.1f", points[0].X, points[0].Y)

	for _, p := range points[1:] {
		fmt.Fprintf(&b, " L %.1f %.1f", p.X, p.Y)
	}

	fmt.Fprintf(
		&s.buf,
		`<path d="%s" fill="none" stroke="%s" stroke-width="%.1f"/>`+"\n",
		b.String(), rgbaToCSS(stroke), strokeWidth,
	)
}

func (s *svgBackend) DrawText(
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

	anchorStr := svgAnchor(anchor)
	escaped := html.EscapeString(text)

	if rotation != 0 {
		deg := rotation * 180.0 / math.Pi

		fmt.Fprintf(
			&s.buf,
			`<text x="%.2f" y="%.2f" fill="%s" font-size="%.1f" font-family="sans-serif" `+
				`text-anchor="%s" dominant-baseline="central" `+
				`transform="rotate(%.2f %.2f %.2f)">%s</text>`+"\n",
			pos.X, pos.Y, rgbaToCSS(ink), fontSize,
			anchorStr, deg, pos.X, pos.Y, escaped,
		)

		return
	}

	fmt.Fprintf(
		&s.buf,
		`<text x="%.2f" y="%.2f" fill="%s" font-size="%.1f" font-family="sans-serif" `+
			`text-anchor="%s" dominant-baseline="central">%s</text>`+"\n",
		pos.X, pos.Y, rgbaToCSS(ink), fontSize, anchorStr, escaped,
	)
}

func (s *svgBackend) DrawArcText(
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

	arcR := radius - 14.0
	if arcR <= 0 {
		return
	}

	pathID := fmt.Sprintf("arc-%d", s.buf.Len())

	// A half-circle arc from the left side to the right side, sweeping
	// clockwise (sweep-flag=1), passes through the top of the circle.
	// With startOffset="50%" and text-anchor="middle", the text is
	// centred at the top.
	fmt.Fprintf(

		&s.buf,
		`<defs><path id="%s" d="M%.2f,%.2f A%.2f,%.2f 0 1,1 %.2f,%.2f" fill="none"/></defs>`+"\n",
		pathID,
		center.X-arcR, center.Y,
		arcR, arcR,
		center.X+arcR, center.Y,
	)

	fmt.Fprintf(
		&s.buf,
		`<text fill="%s" font-size="%.1f" font-family="sans-serif">`+
			`<textPath href="#%s" startOffset="50%%" text-anchor="middle">%s</textPath></text>`+"\n",
		rgbaToCSS(ink), fontSize, pathID, html.EscapeString(text),
	)
}

func (s *svgBackend) Finish(outputPath string) (err error) {
	s.buf.WriteString("</svg>\n")

	f, err := os.Create(outputPath)
	if err != nil {
		return eris.Wrap(err, "failed to create SVG file")
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = eris.Wrap(closeErr, "failed to close SVG file")
		}
	}()

	if _, err := f.Write(s.buf.Bytes()); err != nil {
		return eris.Wrap(err, "failed to write SVG")
	}

	return nil
}

func rgbaToCSS(c color.RGBA) string {
	if c.A == 255 {
		return fmt.Sprintf("rgb(%d,%d,%d)", c.R, c.G, c.B)
	}

	return fmt.Sprintf("rgba(%d,%d,%d,%.3f)", c.R, c.G, c.B, float64(c.A)/255.0)
}

func svgAnchor(a model.TextAnchor) string {
	switch a {
	case model.AnchorMiddle:
		return "middle"
	case model.AnchorEnd:
		return "end"
	default:
		return "start"
	}
}
