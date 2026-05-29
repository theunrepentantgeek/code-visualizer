package canvas

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/legendlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

// legendBuilder collects legend primitives (rectangles, text) for
// decomposition into the Canvas shape list.
type legendBuilder struct {
	shapes   []layeredShape
	order    int
	bgFill   color.RGBA
	bgBorder color.RGBA
	swBorder color.RGBA
	titleInk color.RGBA
	labelInk color.RGBA
}

func newLegendBuilder(baseOrder int) *legendBuilder {
	return &legendBuilder{
		order:    baseOrder,
		bgFill:   color.RGBA{R: 255, G: 255, B: 255, A: 230},
		bgBorder: color.RGBA{R: 153, G: 153, B: 153, A: 204},
		swBorder: color.RGBA{R: 102, G: 102, B: 102, A: 255},
		titleInk: color.RGBA{R: 38, G: 38, B: 38, A: 255},
		labelInk: color.RGBA{R: 51, G: 51, B: 51, A: 255},
	}
}

// decomposeLegend converts the legend configuration into primitive shapes
// (rectangles and text) that can be dispatched through the standard
// Canvas shape pipeline.
func (c *Canvas) decomposeLegend() []layeredShape {
	data := c.legend.toLegendData()
	if data == nil || data.Position == model.LegendPositionNone || len(data.Entries) == 0 {
		return nil
	}

	w, h := legendlayout.MeasureLegend(data, legendlayout.NewBasicMeasurer())
	ox, oy := legendlayout.LegendOrigin(
		data.Position, float64(c.width), float64(c.height), w, h,
	)

	lb := newLegendBuilder(len(c.shapes))
	lb.addBackground(ox, oy, w, h)

	px := ox + model.LegendPadding
	py := oy + model.LegendPadding

	if data.Orientation == model.LegendOrientationHorizontal {
		lb.addEntriesH(data, px, py)
	} else {
		lb.addEntriesV(data, px, py)
	}

	return lb.shapes
}

func (lb *legendBuilder) addBackground(x, y, w, h float64) {
	lb.addRect(x, y, w, h, lb.bgFill, lb.bgBorder, 1.0)
}

func (lb *legendBuilder) addEntriesV(
	data *model.LegendData, x, y float64,
) {
	cy := y

	if data.LabelSample != nil {
		cy = lb.addLabelSample(data.LabelSample, x, cy)
		if len(data.Entries) > 0 {
			cy += model.EntryGap
		}
	}

	for i, entry := range data.Entries {
		if i > 0 {
			cy += model.EntryGap
		}

		cy = lb.addEntry(data.Orientation, entry, x, cy)
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
	// Compute center x for the two-line title: center over the swatch column.
	var contentW float64
	if orientation == model.LegendOrientationHorizontal {
		contentW = legendlayout.MeasureEntryHWidth(entry)
	} else {
		contentW = legendlayout.MeasureEntryVContentWidth(entry)
	}

	centerX := x + contentW/2

	lb.addTextShape(
		centerX, y+model.LegendLineHeight/2,
		entry.Label, lb.titleInk, model.TitleFontSize, AnchorMiddle,
	)
	lb.addTextShape(
		centerX, y+model.LegendLineHeight+model.LegendLineHeight/2,
		entry.Metric, lb.titleInk, model.TitleFontSize, AnchorMiddle,
	)

	y += 2*model.LegendLineHeight + model.LabelGap

	if entry.Kind == model.LegendEntryCategorical {
		return lb.addCategorySwatches(orientation, entry, x, y)
	}

	return lb.addNumericSwatches(orientation, entry, x, y)
}

func (lb *legendBuilder) addNumericSwatches(
	orientation model.LegendOrientation, entry model.LegendEntryData, x, y float64,
) float64 {
	if len(entry.Swatches) == 0 {
		return y
	}

	if orientation == model.LegendOrientationHorizontal {
		return lb.addNumericSwatchesH(entry, x, y)
	}

	return lb.addNumericSwatchesV(entry, x, y)
}

func (lb *legendBuilder) addNumericSwatchesV(
	entry model.LegendEntryData, x, y float64,
) float64 {
	step := model.SwatchSize
	if entry.IsBorder {
		// add gap equal to outline width so adjacent swatch borders are drawn
		// side-by-side rather than overlapping
		step += model.BorderSwatchOutlineWidth
	}

	for _, sw := range entry.Swatches {
		if entry.IsBorder {
			lb.addOutlineSwatch(x, y, sw.Colour)
		} else {
			lb.addSwatch(x, y, sw.Colour)
		}

		if sw.Label != "" {
			lb.addTextShape(
				x+model.SwatchSize+model.LabelGap, y+model.SwatchSize,
				sw.Label, lb.labelInk, model.LegendFontSize, AnchorStart,
			)
		}

		y += step
	}

	return y
}

func (lb *legendBuilder) addNumericSwatchesH(
	entry model.LegendEntryData, x, y float64,
) float64 {
	cx := x

	step := model.SwatchSize
	if entry.IsBorder {
		step += model.BorderSwatchOutlineWidth
	}

	for _, sw := range entry.Swatches {
		x := cx
		if entry.IsBorder {
			lb.addOutlineSwatch(x, y, sw.Colour)
		} else {
			lb.addSwatch(x, y, sw.Colour)
		}

		if sw.Label != "" {
			lb.addTextShape(
				cx+model.SwatchSize, y+model.SwatchSize+model.LegendLineHeight,
				sw.Label, lb.labelInk, model.LegendFontSize, AnchorMiddle,
			)
		}

		cx += step
	}

	return y + model.SwatchSize + model.LegendLineHeight + model.LabelGap
}

func (lb *legendBuilder) addCategorySwatches(
	orientation model.LegendOrientation, entry model.LegendEntryData, x, y float64,
) float64 {
	if orientation == model.LegendOrientationHorizontal {
		return lb.addCategorySwatchesH(entry, x, y)
	}

	return lb.addCategorySwatchesV(entry, x, y)
}

func (lb *legendBuilder) addCategorySwatchesV(
	entry model.LegendEntryData, x, y float64,
) float64 {
	gap := model.SwatchGap
	if entry.IsBorder {
		gap = model.BorderSwatchOutlineWidth
	}

	for _, sw := range entry.Swatches {
		if entry.IsBorder {
			lb.addOutlineSwatch(x, y, sw.Colour)
		} else {
			lb.addSwatch(x, y, sw.Colour)
		}

		lb.addTextShape(
			x+model.SwatchSize+model.LabelGap, y+model.SwatchSize/2,
			sw.Label, lb.labelInk, model.LegendFontSize, AnchorStart,
		)

		y += model.SwatchSize + gap
	}

	return y
}

func (lb *legendBuilder) addCategorySwatchesH(
	entry model.LegendEntryData, x, y float64,
) float64 {
	cx := x

	for _, sw := range entry.Swatches {
		x := cx
		if entry.IsBorder {
			lb.addOutlineSwatch(x, y, sw.Colour)
		} else {
			lb.addSwatch(x, y, sw.Colour)
		}

		lb.addTextShape(
			cx+model.SwatchSize/2, y+model.SwatchSize+model.LegendLineHeight,
			sw.Label, lb.labelInk, model.LegendFontSize, AnchorMiddle,
		)

		cx += legendlayout.MeasureCatSwatchColumnWidth(sw.Label)
	}

	return y + model.SwatchSize + model.LegendLineHeight + model.LabelGap
}

func (lb *legendBuilder) addLabelSample(sample *model.LegendLabelSample, x, y float64) float64 {
	if sample == nil {
		return y
	}

	w, h := legendlayout.MeasureLabelSample(sample)
	if w <= 0 || h <= 0 {
		return y
	}

	lb.addRect(x, y, w, h, white, lb.swBorder, 0.5)

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
			AnchorMiddle,
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
	transparent := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	lb.addRect(x, y, model.SwatchSize, model.SwatchSize, transparent, borderColour, model.BorderSwatchOutlineWidth)
}

func (lb *legendBuilder) addRect(
	x, y, w, h float64, fill, border color.RGBA, borderWidth float64,
) {
	spec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:        FixedInk(fill),
			Border:      FixedInk(border),
			BorderWidth: borderWidth,
		},
	}

	lb.shapes = append(lb.shapes, layeredShape{
		layer: LayerOverlay,
		order: lb.order,
		shape: &Rectangle{Spec: spec, X: x, Y: y, W: w, H: h, Focus: model.Point{X: 0.5, Y: 0.5}},
	})

	lb.order++
}

func (lb *legendBuilder) addTextShape(
	x, y float64, content string, ink color.RGBA,
	fontSize float64, anchor TextAnchor,
) {
	spec := &TextSpec{
		Ink:      FixedInk(ink),
		FontSize: fontSize,
		Anchor:   anchor,
	}

	lb.shapes = append(lb.shapes, layeredShape{
		layer: LayerOverlay,
		order: lb.order,
		shape: &Text{Spec: spec, X: x, Y: y, Content: content},
	})

	lb.order++
}
