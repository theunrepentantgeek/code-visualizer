package inks_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

func TestMeasureValue_SetsMeasureKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mv := inks.MeasureValue(3.14)
	g.Expect(mv.Kind).To(Equal(metric.Measure))
	g.Expect(mv.Measure).To(Equal(3.14))
}

func TestQuantityValue_SetsQuantityKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mv := inks.QuantityValue(42)
	g.Expect(mv.Kind).To(Equal(metric.Quantity))
	g.Expect(mv.Quantity).To(Equal(42))
}

func TestCategoryValue_SetsCategoryKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mv := inks.CategoryValue("go")
	g.Expect(mv.Kind).To(Equal(metric.Classification))
	g.Expect(mv.Category).To(Equal("go"))
}

func TestZeroValue_HasZeroKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var mv inks.MetricValue
	g.Expect(mv.Kind).To(Equal(metric.Quantity))
	g.Expect(mv.Measure).To(Equal(0.0))
	g.Expect(mv.Quantity).To(Equal(0))
	g.Expect(mv.Category).To(BeEmpty())
}
