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
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// defaultFontSize is the font size used when callers pass fontSize <= 0,
// indicating "use the backend's default".
const defaultFontSize = 12.0

type svgBackend struct {
	width       int
	height      int
	buf         bytes.Buffer
	gradID      int
	colourCache map[color.RGBA]string
	gradCache   map[string]string
}

// New creates an SVG backend with the given dimensions.
func New(width, height int) model.Backend {
	b := &svgBackend{
		width:       width,
		height:      height,
		colourCache: make(map[color.RGBA]string),
		gradCache:   make(map[string]string),
	}
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
	pos geometry.Point, size model.Size, fill, border model.Fill, borderWidth float64,
) {
	fillAttr := s.svgFillAttr(fill)
	borderColour := model.SolidColor(border)

	fmt.Fprintf(
		&s.buf,
		`<rect x="%.3f" y="%.3f" width="%.3f" height="%.3f" fill="%s" stroke="%s" stroke-width="%.3f"/>`+"\n",
		pos.X, pos.Y, size.Width, size.Height,
		fillAttr, s.colourCSS(borderColour), borderWidth,
	)
}

// emitRadialGradient emits the gradient <defs> block on its first call for a
// given gradient specification, and returns a "url(#id)" fill-attribute string
// on every call (including cache hits).  Callers can use the returned string
// directly as the fill attribute without additional formatting.
func (s *svgBackend) emitRadialGradient(grad model.RadialGradientFill) string {
	centerCSS := s.colourCSS(grad.Center)
	edgeCSS := s.colourCSS(grad.Edge)
	focus := geometry.Point{X: grad.Focus.X * 100, Y: grad.Focus.Y * 100}

	key := fmt.Sprintf(
		"%s|%s|%.3f|%.3f",
		centerCSS, edgeCSS,
		focus.X, focus.Y,
	)

	if urlRef, ok := s.gradCache[key]; ok {
		return urlRef
	}

	s.gradID++
	id := fmt.Sprintf("rg%d", s.gradID)
	urlRef := "url(#" + id + ")"

	// A 70% radius reaches the rectangle edges while avoiding corner emphasis.
	fmt.Fprintf(
		&s.buf,
		`<defs><radialGradient id="%s" cx="50%%" cy="50%%" r="70%%" fx="%.3f%%" fy="%.3f%%">`+
			`<stop offset="0%%" stop-color="%s"/>`+
			`<stop offset="100%%" stop-color="%s"/>`+
			`</radialGradient></defs>`+"\n",
		id,
		focus.X, focus.Y,
		centerCSS, edgeCSS,
	)

	s.gradCache[key] = urlRef

	return urlRef
}

func (s *svgBackend) DrawDisc(
	center geometry.Point, radius float64, fill, border model.Fill, borderWidth float64,
) {
	fillAttr := s.svgFillAttr(fill)
	borderColour := model.SolidColor(border)

	fmt.Fprintf(
		&s.buf,
		`<circle cx="%.3f" cy="%.3f" r="%.3f" fill="%s" stroke="%s" stroke-width="%.3f"/>`+"\n",
		center.X, center.Y, radius,
		fillAttr, s.colourCSS(borderColour), borderWidth,
	)
}

func (s *svgBackend) DrawPolygon(
	points []geometry.Point, fill, border model.Fill, borderWidth float64,
) {
	if len(points) < 3 {
		return
	}

	var pointPairs strings.Builder

	for i, point := range points {
		if i > 0 {
			pointPairs.WriteByte(' ')
		}

		fmt.Fprintf(&pointPairs, "%.3f,%.3f", point.X, point.Y)
	}

	fmt.Fprintf(&s.buf, `<polygon points="%s" fill="%s"`, pointPairs.String(), s.svgFillAttr(fill))

	if borderWidth > 0 {
		fmt.Fprintf(&s.buf, ` stroke="%s" stroke-width="%.3f"`, s.colourCSS(model.SolidColor(border)), borderWidth)
	}

	s.buf.WriteString("/>\n")
}

func (s *svgBackend) DrawFilledPath(loops [][]geometry.Point, fill color.RGBA) {
	var pathData strings.Builder

	// Closed loops are combined with even-odd filling so holes and islands remain intact.
	for _, loop := range loops {
		if len(loop) < 3 {
			continue
		}

		if pathData.Len() > 0 {
			pathData.WriteByte(' ')
		}

		fmt.Fprintf(&pathData, "M %.3f %.3f", loop[0].X, loop[0].Y)

		for _, point := range loop[1:] {
			fmt.Fprintf(&pathData, " L %.3f %.3f", point.X, point.Y)
		}

		pathData.WriteString(" Z")
	}

	if pathData.Len() == 0 {
		return
	}

	fmt.Fprintf(
		&s.buf,
		`<path d="%s" fill="%s" fill-rule="evenodd"/>`+"\n",
		pathData.String(), s.colourCSS(fill),
	)
}

func (s *svgBackend) DrawLine(from, to geometry.Point, stroke color.RGBA, strokeWidth float64) {
	fmt.Fprintf(
		&s.buf,
		`<line x1="%.3f" y1="%.3f" x2="%.3f" y2="%.3f" stroke="%s" stroke-width="%.3f"/>`+"\n",
		from.X, from.Y, to.X, to.Y,
		s.colourCSS(stroke), strokeWidth,
	)
}

func (s *svgBackend) DrawPath(points []geometry.Point, stroke color.RGBA, strokeWidth float64) {
	if len(points) < 2 {
		return
	}

	// Write the path data directly into s.buf to avoid an intermediate
	// strings.Builder allocation and the subsequent copy into s.buf.
	// For spiral tracks (500–1500+ points) this saves ~6–22 KB of allocation.
	fmt.Fprintf(&s.buf, `<path d="M %.3f %.3f`, points[0].X, points[0].Y)

	for _, p := range points[1:] {
		fmt.Fprintf(&s.buf, ` L %.3f %.3f`, p.X, p.Y)
	}

	fmt.Fprintf(&s.buf, `" fill="none" stroke="%s" stroke-width="%.3f"/>`+"\n",
		s.colourCSS(stroke), strokeWidth)
}

func (s *svgBackend) DrawText(
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

	anchorStr := svgAnchor(anchor)
	escaped := html.EscapeString(text)

	if rotation != 0 {
		deg := rotation * 180.0 / math.Pi

		fmt.Fprintf(
			&s.buf,
			`<text x="%.3f" y="%.3f" fill="%s" font-size="%.3f" font-family="sans-serif" `+
				`text-anchor="%s" dominant-baseline="central" `+
				`transform="rotate(%.3f %.3f %.3f)">%s</text>`+"\n",
			pos.X, pos.Y, s.colourCSS(ink), fontSize,
			anchorStr, deg, pos.X, pos.Y, escaped,
		)

		return
	}

	fmt.Fprintf(
		&s.buf,
		`<text x="%.3f" y="%.3f" fill="%s" font-size="%.3f" font-family="sans-serif" `+
			`text-anchor="%s" dominant-baseline="central">%s</text>`+"\n",
		pos.X, pos.Y, s.colourCSS(ink), fontSize, anchorStr, escaped,
	)
}

func (s *svgBackend) DrawArcText(
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

	arcR := radius - model.ArcTextInset
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
		`<defs><path id="%s" d="M%.3f,%.3f A%.3f,%.3f 0 0,1 %.3f,%.3f" fill="none"/></defs>`+"\n",
		pathID,
		center.X-arcR, center.Y,
		arcR, arcR,
		center.X+arcR, center.Y,
	)

	fmt.Fprintf(
		&s.buf,
		`<text fill="%s" font-size="%.3f" font-family="sans-serif" dominant-baseline="middle">`+
			`<textPath href="#%s" startOffset="50%%" text-anchor="middle">%s</textPath></text>`+"\n",
		s.colourCSS(ink), fontSize, pathID, html.EscapeString(text),
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

// svgFillAttr returns the SVG fill attribute value for the given fill.
// For gradients it emits the gradient definition and returns a url(#id) ref;
// for solid or default fills it returns a CSS colour string.
func (s *svgBackend) svgFillAttr(fill model.Fill) string {
	if f, ok := fill.(model.RadialGradientFill); ok {
		return s.emitRadialGradient(f)
	}

	return s.colourCSS(model.SolidColor(fill))
}

// colourCSS returns the CSS colour string for c, using a per-backend cache to
// avoid repeated fmt.Sprintf allocations for the same colour. In typical
// visualizations, nodes share a small number of palette colours (e.g. 16
// buckets), so the cache hit rate is high.
func (s *svgBackend) colourCSS(c color.RGBA) string {
	if cached, ok := s.colourCache[c]; ok {
		return cached
	}

	result := rgbaToCSS(c)
	s.colourCache[c] = result

	return result
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
