package legend

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/legendlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// RenderInto adds the legend overlay shapes to cv at LayerOverlay.
// Does nothing when cfg is nil, has no entries, or is positioned None.
func RenderInto(cv *canvas.Canvas, cfg *Config) {
	if cfg == nil {
		return
	}

	data := cfg.toLegendData()
	if data == nil || data.Position == model.LegendPositionNone || len(data.Entries) == 0 {
		return
	}

	w, h := legendlayout.MeasureLegend(data, legendlayout.NewBasicMeasurer())
	ox, oy := legendOrigin(cv, data.Position, w, h)

	lb := newLegendBuilder(cv)
	lb.addBackground(ox, oy, w, h)

	px := ox + model.LegendPadding
	py := oy + model.LegendPadding

	if data.Orientation == model.LegendOrientationHorizontal {
		lb.addEntriesH(data, px, py)
	} else {
		contentAreaW := w - 2*model.LegendPadding
		lb.addEntriesV(data, px, py, contentAreaW)
	}
}

// legendOrigin computes the top-left (x, y) of the legend, respecting the
// drawing bounds for top-center and bottom-center positions so that the
// legend doesn't overlap the title or footer.
func legendOrigin(
	cv *canvas.Canvas, position model.LegendPosition, legendW, legendH float64,
) (ox, oy float64) {
	m := model.LegendMargin
	cw := float64(cv.Width())
	ch := float64(cv.Height())

	switch position {
	case model.LegendPositionTopCenter:
		return (cw - legendW) / 2, float64(cv.DrawingMinY()) + m
	case model.LegendPositionBottomCenter:
		return (cw - legendW) / 2, float64(cv.DrawingMaxY()) - legendH - m
	default:
		return legendlayout.LegendOrigin(position, cw, ch, legendW, legendH)
	}
}

// Default colours used by legendBuilder. They are copied into each
// builder instance so that a future PR can introduce per-render overrides
// without touching the call sites.
//
//nolint:gochecknoglobals // package-level colour defaults
var (
	defaultBgFill   = color.RGBA{R: 255, G: 255, B: 255, A: 230}
	defaultBgBorder = color.RGBA{R: 153, G: 153, B: 153, A: 204}
	defaultSwBorder = color.RGBA{R: 102, G: 102, B: 102, A: 255}
	defaultTitleInk = color.RGBA{R: 38, G: 38, B: 38, A: 255}
	defaultLabelInk = color.RGBA{R: 51, G: 51, B: 51, A: 255}
)

// legendBuilder writes legend primitives (rectangles, text) directly to
// the canvas at LayerOverlay.
type legendBuilder struct {
	cv       *canvas.Canvas
	bgFill   color.RGBA
	bgBorder color.RGBA
	swBorder color.RGBA
	titleInk color.RGBA
	labelInk color.RGBA
}

func newLegendBuilder(cv *canvas.Canvas) *legendBuilder {
	return &legendBuilder{
		cv:       cv,
		bgFill:   defaultBgFill,
		bgBorder: defaultBgBorder,
		swBorder: defaultSwBorder,
		titleInk: defaultTitleInk,
		labelInk: defaultLabelInk,
	}
}

func (lb *legendBuilder) addBackground(x, y, w, h float64) {
	lb.addRect(x, y, w, h, lb.bgFill, lb.bgBorder, 1.0)
}

func (lb *legendBuilder) addEntriesV(
	data *model.LegendData, x, y float64, contentAreaW float64,
) {
	cy := y

	if data.LabelSample != nil {
		sampleW, _ := legendlayout.MeasureLabelSample(data.LabelSample)
		sampleX := x + (contentAreaW-sampleW)/2
		cy = lb.addLabelSample(data.LabelSample, sampleX, cy)

		if len(data.Entries) > 0 {
			cy += model.EntryGap
		}
	}

	entryX := x + legendlayout.ContentOffsetV(data)

	for i, entry := range data.Entries {
		if i > 0 {
			cy += model.EntryGap
		}

		cy = lb.addEntry(data.Orientation, entry, entryX, cy)
	}
}

func (lb *legendBuilder) addEntriesH(
	data *model.LegendData, x, y float64,
) {
	cx := x

	if data.LabelSample != nil {
		sampleW, _ := legendlayout.MeasureLabelSample(data.LabelSample)
		lb.addLabelSample(data.LabelSample, cx, y)

		cx += sampleW
		if len(data.Entries) > 0 {
			cx += model.EntryGap
		}
	}

	for i, entry := range data.Entries {
		if i > 0 {
			cx += model.EntryGap
		}

		ew := legendlayout.MeasureEntryHWidth(entry)
		lb.addEntry(data.Orientation, entry, cx, y)
		cx += ew
	}
}

func (lb *legendBuilder) addEntry(
	orientation model.LegendOrientation, entry model.LegendEntryData, x, y float64,
) float64 {
	var contentW float64
	if orientation == model.LegendOrientationHorizontal {
		contentW = legendlayout.MeasureEntryHWidth(entry)
	} else {
		contentW = legendlayout.MeasureEntryVContentWidth(entry)
	}

	centerX := x + contentW/2

	lb.addTextShape(
		centerX, y+model.LegendLineHeight/2,
		entry.Label, lb.titleInk, model.TitleFontSize, canvas.AnchorMiddle,
	)
	lb.addTextShape(
		centerX, y+model.LegendLineHeight+model.LegendLineHeight/2,
		entry.Metric, lb.titleInk, model.TitleFontSize, canvas.AnchorMiddle,
	)

	y += 2*model.LegendLineHeight + model.LabelGap

	if entry.Kind == model.LegendEntryCategorical {
		return lb.addCategorySwatches(orientation, entry, x, y)
	}

	return lb.addNumericSwatches(orientation, entry, x, y)
}

// swatchCursor tracks the position of the next swatch in a legend strip.
// For vertical orientation it advances along Y; for horizontal, along X.
type swatchCursor struct {
	x, y       float64
	horizontal bool
}

// swatchPos returns the top-left corner of the current swatch.
func (c *swatchCursor) swatchPos() (x, y float64) { return c.x, c.y }

// numericLabelPos returns the position and anchor for a numeric swatch label.
// Vertical: to the right of the swatch. Horizontal: below the swatch.
func (c *swatchCursor) numericLabelPos() (x, y float64, anchor canvas.TextAnchor) {
	if c.horizontal {
		return c.x + model.SwatchSize, c.y + model.SwatchSize + model.LegendLineHeight, canvas.AnchorMiddle
	}

	return c.x + model.SwatchSize + model.LabelGap, c.y + model.SwatchSize, canvas.AnchorStart
}

// catLabelPos returns the position and anchor for a categorical swatch label.
// Vertical: to the right, vertically centred on the swatch.
// Horizontal: below the swatch, centred horizontally.
func (c *swatchCursor) catLabelPos() (x, y float64, anchor canvas.TextAnchor) {
	if c.horizontal {
		return c.x + model.SwatchSize/2, c.y + model.SwatchSize + model.LegendLineHeight, canvas.AnchorMiddle
	}

	return c.x + model.SwatchSize + model.LabelGap, c.y + model.SwatchSize/2, canvas.AnchorStart
}

// advance moves the cursor by delta along the main axis.
func (c *swatchCursor) advance(delta float64) {
	if c.horizontal {
		c.x += delta
	} else {
		c.y += delta
	}
}

// endY returns the Y coordinate that marks the end of this swatch block.
// Vertical: the cursor's current Y, already advanced past the last swatch.
// Horizontal: startY plus the fixed block height (swatches don't change Y).
func (c *swatchCursor) endY(startY float64) float64 {
	if c.horizontal {
		return startY + model.SwatchSize + model.LegendLineHeight + model.LabelGap
	}

	return c.y
}

func (lb *legendBuilder) addNumericSwatches(
	orientation model.LegendOrientation, entry model.LegendEntryData, x, y float64,
) float64 {
	if len(entry.Swatches) == 0 {
		return y
	}

	step := model.SwatchSize
	if entry.IsBorder {
		step += model.BorderSwatchOutlineWidth
	}

	cur := swatchCursor{x: x, y: y, horizontal: orientation == model.LegendOrientationHorizontal}

	for _, sw := range entry.Swatches {
		sx, sy := cur.swatchPos()
		if entry.IsBorder {
			lb.addOutlineSwatch(sx, sy, sw.Colour)
		} else {
			lb.addSwatch(sx, sy, sw.Colour)
		}

		if sw.Label != "" {
			lx, ly, anchor := cur.numericLabelPos()
			lb.addTextShape(lx, ly, sw.Label, lb.labelInk, model.LegendFontSize, anchor)
		}

		cur.advance(step)
	}

	return cur.endY(y)
}

func (lb *legendBuilder) addCategorySwatches(
	orientation model.LegendOrientation, entry model.LegendEntryData, x, y float64,
) float64 {
	if len(entry.Swatches) == 0 {
		return y
	}

	gap := model.SwatchGap
	if entry.IsBorder {
		gap = model.BorderSwatchOutlineWidth
	}

	cur := swatchCursor{x: x, y: y, horizontal: orientation == model.LegendOrientationHorizontal}

	for _, sw := range entry.Swatches {
		sx, sy := cur.swatchPos()
		if entry.IsBorder {
			lb.addOutlineSwatch(sx, sy, sw.Colour)
		} else {
			lb.addSwatch(sx, sy, sw.Colour)
		}

		lx, ly, anchor := cur.catLabelPos()
		lb.addTextShape(lx, ly, sw.Label, lb.labelInk, model.LegendFontSize, anchor)

		if cur.horizontal {
			cur.advance(legendlayout.MeasureCatSwatchColumnWidth(sw.Label))
		} else {
			cur.advance(model.SwatchSize + gap)
		}
	}

	return cur.endY(y)
}

func (lb *legendBuilder) addLabelSample(sample *model.LegendLabelSample, x, y float64) float64 {
	if sample == nil {
		return y
	}

	w, h := legendlayout.MeasureLabelSample(sample)
	if w <= 0 || h <= 0 {
		return y
	}

	lb.addRect(x, y, w, h, palette.White, lb.swBorder, 0.5)

	centerX := x + w/2
	totalH := float64(len(sample.Lines)) * model.LegendLineHeight
	startY := y + (h-totalH)/2 + model.LegendLineHeight/2

	for i, line := range sample.Lines {
		lb.addTextShape(
			centerX,
			startY+float64(i)*model.LegendLineHeight,
			line,
			lb.labelInk,
			model.LegendFontSize,
			canvas.AnchorMiddle,
		)
	}

	return y + h
}

func (lb *legendBuilder) addSwatch(x, y float64, fill color.RGBA) {
	lb.addRect(x, y, model.SwatchSize, model.SwatchSize, fill, lb.swBorder, 0.5)
}

// addOutlineSwatch renders a swatch as a coloured outline with a white interior,
// to represent a border metric rather than a fill metric.
func (lb *legendBuilder) addOutlineSwatch(x, y float64, borderColour color.RGBA) {
	lb.addRect(x, y, model.SwatchSize, model.SwatchSize, palette.White, borderColour, model.BorderSwatchOutlineWidth)
}

func (lb *legendBuilder) addRect(
	x, y, w, h float64, fill, border color.RGBA, borderWidth float64,
) {
	spec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(fill),
			Border:      inks.FixedInk(border),
			BorderWidth: borderWidth,
		},
	}

	lb.cv.AddRectangle(canvas.LayerOverlay, canvas.Rectangle{
		Spec: spec, X: x, Y: y, W: w, H: h, Focus: model.Point{X: 0.5, Y: 0.5},
	})
}

func (lb *legendBuilder) addTextShape(
	x, y float64, content string, ink color.RGBA,
	fontSize float64, anchor canvas.TextAnchor,
) {
	spec := &canvas.TextSpec{
		Ink:      inks.FixedInk(ink),
		FontSize: fontSize,
		Anchor:   anchor,
	}

	lb.cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec: spec, X: x, Y: y, Content: content,
	})
}
