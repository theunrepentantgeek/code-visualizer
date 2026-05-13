package main

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

func TestKindLabel_Quantity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(kindLabel(metric.Quantity)).To(Equal("quantity"))
}

func TestKindLabel_Measure(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(kindLabel(metric.Measure)).To(Equal("measure"))
}

func TestKindLabel_Classification(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(kindLabel(metric.Classification)).To(Equal("category"))
}

func TestKindLabel_Unknown(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(kindLabel(metric.Kind(99))).To(Equal("unknown"))
}
