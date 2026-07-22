package legendlayout

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

// MeasureCatSwatchColumnWidth

func TestMeasureCatSwatchColumnWidth_ShortLabel_UsesSwatchSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "go" is 2 chars wide (14px with 7px/char) — narrower than SwatchSize (28).
	// Width = max(SwatchSize, labelW) + SwatchGap + LabelGap = 28 + 4 + 6 = 38.
	w := MeasureCatSwatchColumnWidth("go")
	g.Expect(w).To(BeNumerically("~", 38, 1))
}

func TestMeasureCatSwatchColumnWidth_LongLabel_UsesLabelWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "javascript" is 10 chars (70px) — wider than SwatchSize (28).
	// Width = max(28, 70) + 4 + 6 = 80.
	w := MeasureCatSwatchColumnWidth("javascript")
	g.Expect(w).To(BeNumerically("~", 80, 2))
}

func TestMeasureCatSwatchColumnWidth_EmptyLabel_ReturnsMinimum(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Empty label: max(28, 0) + 4 + 6 = 38.
	w := MeasureCatSwatchColumnWidth("")
	g.Expect(w).To(BeNumerically("~", 38, 1))
}

// MeasureLabelSample

func TestMeasureLabelSample_Nil_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	w, h := MeasureLabelSample(nil)
	g.Expect(w).To(Equal(0.0))
	g.Expect(h).To(Equal(0.0))
}

func TestMeasureLabelSample_EmptyLines_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	w, h := MeasureLabelSample(&model.LegendLabelSample{Lines: nil})
	g.Expect(w).To(Equal(0.0))
	g.Expect(h).To(Equal(0.0))
}

func TestMeasureLabelSample_ShortLine_ReturnsSquareAtLeastDoubleSwatchSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "ab" is 14px wide; textH = 1 * LegendLineHeight = 16.
	// side = max(SwatchSize*2, 14+2*LabelGap, 16+2*LabelGap) = max(56, 26, 28) = 56.
	w, h := MeasureLabelSample(&model.LegendLabelSample{Lines: []string{"ab"}})
	g.Expect(w).To(BeNumerically("~", 56, 1))
	g.Expect(h).To(Equal(w)) // result is a square
}

func TestMeasureLabelSample_LongLine_SquareDrivenByTextWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// 30 chars × 7px = 210px; side = max(56, 210+12, 16+12) = 222.
	long := "123456789012345678901234567890"
	w, h := MeasureLabelSample(&model.LegendLabelSample{Lines: []string{long}})
	g.Expect(w).To(BeNumerically(">", model.SwatchSize*2))
	g.Expect(h).To(Equal(w))
}

func TestMeasureLabelSample_MultipleLines_SquareDrivenByHeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// 5 short lines: textH = 5 * 16 = 80; side = max(56, short+12, 80+12) = 92.
	lines := []string{"a", "b", "c", "d", "e"}
	w, h := MeasureLabelSample(&model.LegendLabelSample{Lines: lines})
	g.Expect(w).To(BeNumerically(">", model.SwatchSize*2))
	g.Expect(h).To(Equal(w))
}

// MeasureEntryHWidth

func TestMeasureEntryHWidth_NumericEntry_ReturnsPositiveWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := model.LegendEntryData{
		Label:  "Fill",
		Metric: "file-size",
		Kind:   model.LegendEntryNumeric,
		Swatches: []model.LegendSwatch{
			{Colour: color.RGBA{R: 50, A: 255}, Label: "100"},
			{Colour: color.RGBA{R: 150, A: 255}, Label: "500"},
			{Colour: color.RGBA{R: 250, A: 255}, Label: ""},
		},
	}

	w := MeasureEntryHWidth(entry)
	// 3 swatches × SwatchSize = 84; title "file-size" = 63px → max = 84.
	g.Expect(w).To(BeNumerically(">", 0))
	g.Expect(w).To(BeNumerically("~", 84, 2))
}

func TestMeasureEntryHWidth_CategoricalEntry_ReturnsPositiveWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := model.LegendEntryData{
		Label:  "Border",
		Metric: "file-type",
		Kind:   model.LegendEntryCategorical,
		Swatches: []model.LegendSwatch{
			{Colour: color.RGBA{R: 0, G: 173, B: 216, A: 255}, Label: "go"},
			{Colour: color.RGBA{R: 222, G: 165, B: 132, A: 255}, Label: "py"},
		},
	}

	w := MeasureEntryHWidth(entry)
	// 2 categories × (max(28,14)+4+6) = 2 × 38 = 76; title "file-type" = 63 → max = 76.
	g.Expect(w).To(BeNumerically(">", 0))
	g.Expect(w).To(BeNumerically("~", 76, 2))
}

// MeasureEntryVContentWidth

func TestMeasureEntryVContentWidth_NumericEntry_ReturnsPositiveWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := model.LegendEntryData{
		Label:  "Fill",
		Metric: "file-lines",
		Kind:   model.LegendEntryNumeric,
		Swatches: []model.LegendSwatch{
			{Colour: color.RGBA{R: 50, A: 255}, Label: "10"},
			{Colour: color.RGBA{R: 150, A: 255}, Label: "50"},
			{Colour: color.RGBA{R: 250, A: 255}, Label: ""},
		},
	}

	w := MeasureEntryVContentWidth(entry)
	// "10" → bw = 28 + 6 + 14 = 48; w = 48.
	g.Expect(w).To(BeNumerically(">", 0))
	g.Expect(w).To(BeNumerically("~", 48, 2))
}

func TestMeasureEntryVContentWidth_CategoricalEntry_ReturnsPositiveWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := model.LegendEntryData{
		Label:  "Fill",
		Metric: "file-type",
		Kind:   model.LegendEntryCategorical,
		Swatches: []model.LegendSwatch{
			{Colour: color.RGBA{R: 0, A: 255}, Label: "go"},
			{Colour: color.RGBA{R: 128, A: 255}, Label: "py"},
			{Colour: color.RGBA{R: 200, A: 255}, Label: "rs"},
		},
	}

	w := MeasureEntryVContentWidth(entry)
	// Each label "go"/"py"/"rs" = 14px → cw = 28+6+14 = 48; w = 48.
	g.Expect(w).To(BeNumerically(">", 0))
	g.Expect(w).To(BeNumerically("~", 48, 2))
}
