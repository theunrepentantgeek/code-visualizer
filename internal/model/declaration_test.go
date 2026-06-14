package model_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

func TestDeclaration_SetAndGetMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d := &model.Declaration{
		Name:       "HandleRequest",
		Kind:       "function",
		Visibility: "public",
	}

	d.SetQuantity("cyclomatic-complexity", 5)
	v, ok := d.Quantity("cyclomatic-complexity")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(5)))
}

func TestDeclaration_MatchesFilter_Public(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d := &model.Declaration{
		Name:       "HandleRequest",
		Kind:       "function",
		Visibility: "public",
	}

	g.Expect(d.MatchesFilter(metric.FilterName("public"))).To(BeTrue())
	g.Expect(d.MatchesFilter(metric.FilterName("private"))).To(BeFalse())
}

func TestDeclaration_MatchesFilter_Private(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d := &model.Declaration{
		Name:       "handleRequest",
		Kind:       "function",
		Visibility: "private",
	}

	g.Expect(d.MatchesFilter(metric.FilterName("private"))).To(BeTrue())
	g.Expect(d.MatchesFilter(metric.FilterName("public"))).To(BeFalse())
}
