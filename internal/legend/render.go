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

	layout := fitLegendToCanvas(cv, data)
	ox, oy := legendOrigin(cv, layout.data.Position, layout.width, layout.height)

	lb := newLegendBuilder(cv, layout.scale)
	lb.addBackground(ox, oy, layout.width, layout.height)

	px := ox + lb.scaleValue(model.LegendPadding)
	py := oy + lb.scaleValue(model.LegendPadding)

	if layout.data.Orientation == model.LegendOrientationHorizontal {
		lb.addEntriesH(layout.data, px, py)
	} else {
		contentAreaW := layout.width - 2*lb.scaleValue(model.LegendPadding)
		lb.addEntriesV(layout.data, px, py, contentAreaW)
	}
}

type legendLayout struct {
	data          *model.LegendData
	width, height float64
	scale         float64
}

// fitLegendToCanvas scales oversized legends to keep them fully visible while
// preserving their configured orientation.
func fitLegendToCanvas(cv *canvas.Canvas, data *model.LegendData) legendLayout {
	w, h := legendlayout.MeasureLegend(data, legendlayout.NewBasicMeasurer())
	if legendFits(cv, w, h) {
		return legendLayout{data: data, width: w, height: h, scale: 1}
	}

	scale := min(
		float64(cv.Width())/w,
		float64(cv.DrawingMaxY()-cv.DrawingMinY())/h,
	)

	return legendLayout{
		data: data, width: w * scale, height: h * scale, scale: scale,
	}
}

func legendFits(cv *canvas.Canvas, legendW, legendH float64) bool {
	return legendW <= float64(cv.Width()) &&
		legendH <= float64(cv.DrawingMaxY()-cv.DrawingMinY())
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
		ox, oy = (cw-legendW)/2, float64(cv.DrawingMinY())+m
	case model.LegendPositionBottomCenter:
		ox, oy = (cw-legendW)/2, float64(cv.DrawingMaxY())-legendH-m
	default:
		ox, oy = legendlayout.LegendOrigin(position, cw, ch, legendW, legendH)
	}

	if legendW <= cw {
		ox = min(max(ox, 0), cw-legendW)
	}

	drawingMinY := float64(cv.DrawingMinY())

	drawingMaxY := float64(cv.DrawingMaxY())
	if legendH <= drawingMaxY-drawingMinY {
		oy = min(max(oy, drawingMinY), drawingMaxY-legendH)
	}

	return ox, oy
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
	scale    float64
	bgFill   color.RGBA
	bgBorder color.RGBA
	swBorder color.RGBA
	titleInk color.RGBA
	labelInk color.RGBA
}

func newLegendBuilder(cv *canvas.Canvas, scale float64) *legendBuilder {
	return &legendBuilder{
		cv:       cv,
		scale:    scale,
		bgFill:   defaultBgFill,
		bgBorder: defaultBgBorder,
		swBorder: defaultSwBorder,
		titleInk: defaultTitleInk,
		labelInk: defaultLabelInk,
	}
}

func (lb *legendBuilder) scaleValue(value float64) float64 {
	return value * lb.scale
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
		sampleX := x + (contentAreaW-lb.scaleValue(sampleW))/2
		cy = lb.addLabelSample(data.LabelSample, sampleX, cy)

		if len(data.Entries) > 0 {
			cy += lb.scaleValue(model.EntryGap)
		}
	}

	entryX := x + lb.scaleValue(legendlayout.ContentOffsetV(data))

	for i, entry := range data.Entries {
		if i > 0 {
			cy += lb.scaleValue(model.EntryGap)
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

		cx += lb.scaleValue(sampleW)
		if len(data.Entries) > 0 {
			cx += lb.scaleValue(model.EntryGap)
		}
	}

	for i, entry := range data.Entries {
		if i > 0 {
			cx += lb.scaleValue(model.EntryGap)
		}

		ew := legendlayout.MeasureEntryHWidth(entry)
		lb.addEntry(data.Orientation, entry, cx, y)
		cx += lb.scaleValue(ew)
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

	centerX := x + lb.scaleValue(contentW)/2

	lb.addTextShape(
		centerX, y+lb.scaleValue(model.LegendLineHeight)/2,
		entry.Label, lb.titleInk, lb.scaleValue(model.TitleFontSize), canvas.AnchorMiddle,
	)
	lb.addTextShape(
		centerX, y+lb.scaleValue(model.LegendLineHeight)+lb.scaleValue(model.LegendLineHeight)/2,
		entry.Metric, lb.titleInk, lb.scaleValue(model.TitleFontSize), canvas.AnchorMiddle,
	)

	y += lb.scaleValue(2*model.LegendLineHeight + model.LabelGap)

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
	scale      float64
}

// swatchPos returns the top-left corner of the current swatch.
func (c *swatchCursor) swatchPos() (x, y float64) { return c.x, c.y }

// numericLabelPos returns the position and anchor for a numeric swatch label.
// Vertical: to the right of the swatch. Horizontal: below the swatch.
func (c *swatchCursor) numericLabelPos() (x, y float64, anchor canvas.TextAnchor) {
	if c.horizontal {
		return c.x + c.scale*model.SwatchSize,
			c.y + c.scale*(model.SwatchSize+model.LegendLineHeight),
			canvas.AnchorMiddle
	}

	return c.x + c.scale*(model.SwatchSize+model.LabelGap),
		c.y + c.scale*model.SwatchSize,
		canvas.AnchorStart
}

// catLabelPos returns the position and anchor for a categorical swatch label.
// Vertical: to the right, vertically centred on the swatch.
// Horizontal: below the swatch, centred horizontally.
func (c *swatchCursor) catLabelPos() (x, y float64, anchor canvas.TextAnchor) {
	if c.horizontal {
		return c.x + c.scale*model.SwatchSize/2,
			c.y + c.scale*(model.SwatchSize+model.LegendLineHeight),
			canvas.AnchorMiddle
	}

	return c.x + c.scale*(model.SwatchSize+model.LabelGap),
		c.y + c.scale*model.SwatchSize/2,
		canvas.AnchorStart
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
		return startY + c.scale*(model.SwatchSize+model.LegendLineHeight+model.LabelGap)
	}

	return c.y
}

func (lb *legendBuilder) addNumericSwatches(
	orientation model.LegendOrientation, entry model.LegendEntryData, x, y float64,
) float64 {
	if len(entry.Swatches) == 0 {
		return y
	}

	step := lb.scaleValue(model.SwatchSize)
	if entry.IsBorder {
		step += lb.scaleValue(model.BorderSwatchOutlineWidth)
	}

	cur := swatchCursor{x: x, y: y, horizontal: orientation == model.LegendOrientationHorizontal, scale: lb.scale}

	for _, sw := range entry.Swatches {
		sx, sy := cur.swatchPos()
		if entry.IsBorder {
			lb.addOutlineSwatch(sx, sy, sw.Colour)
		} else {
			lb.addSwatch(sx, sy, sw.Colour)
		}

		if sw.Label != "" {
			lx, ly, anchor := cur.numericLabelPos()
			lb.addTextShape(lx, ly, sw.Label, lb.labelInk, lb.scaleValue(model.LegendFontSize), anchor)
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

	gap := lb.scaleValue(model.SwatchGap)
	if entry.IsBorder {
		gap = lb.scaleValue(model.BorderSwatchOutlineWidth)
	}

	cur := swatchCursor{x: x, y: y, horizontal: orientation == model.LegendOrientationHorizontal, scale: lb.scale}

	for _, sw := range entry.Swatches {
		sx, sy := cur.swatchPos()
		if entry.IsBorder {
			lb.addOutlineSwatch(sx, sy, sw.Colour)
		} else {
			lb.addSwatch(sx, sy, sw.Colour)
		}

		lx, ly, anchor := cur.catLabelPos()
		lb.addTextShape(lx, ly, sw.Label, lb.labelInk, lb.scaleValue(model.LegendFontSize), anchor)

		if cur.horizontal {
			cur.advance(lb.scaleValue(legendlayout.MeasureCatSwatchColumnWidth(sw.Label)))
		} else {
			cur.advance(lb.scaleValue(model.SwatchSize) + gap)
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

	w = lb.scaleValue(w)
	h = lb.scaleValue(h)

	if sample.Shape == model.LegendLabelSampleCircle {
		spec := &canvas.DiscSpec{
			ShapeStyle: canvas.ShapeStyle{
				Fill:        inks.FixedInk(palette.White),
				Border:      inks.FixedInk(lb.swBorder),
				BorderWidth: lb.scaleValue(0.5),
			},
		}
		lb.cv.AddDisc(canvas.LayerOverlay, canvas.Disc{
			Spec: spec, X: x + w/2, Y: y + h/2, Radius: min(w, h) / 2,
		})
	} else {
		lb.addRect(x, y, w, h, palette.White, lb.swBorder, lb.scaleValue(0.5))
	}

	centerX := x + w/2
	totalH := float64(len(sample.Lines)) * lb.scaleValue(model.LegendLineHeight)
	startY := y + (h-totalH)/2 + lb.scaleValue(model.LegendLineHeight)/2

	for i, line := range sample.Lines {
		lb.addTextShape(
			centerX,
			startY+float64(i)*lb.scaleValue(model.LegendLineHeight),
			line,
			lb.labelInk,
			lb.scaleValue(model.LegendFontSize),
			canvas.AnchorMiddle,
		)
	}

	return y + h
}

func (lb *legendBuilder) addSwatch(x, y float64, fill color.RGBA) {
	size := lb.scaleValue(model.SwatchSize)
	lb.addRect(x, y, size, size, fill, lb.swBorder, lb.scaleValue(0.5))
}

// addOutlineSwatch renders a swatch as a coloured outline with a white interior,
// to represent a border metric rather than a fill metric.
func (lb *legendBuilder) addOutlineSwatch(x, y float64, borderColour color.RGBA) {
	size := lb.scaleValue(model.SwatchSize)
	lb.addRect(x, y, size, size, palette.White, borderColour, lb.scaleValue(model.BorderSwatchOutlineWidth))
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
