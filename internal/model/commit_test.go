package model_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

func TestCommit_SetAndGetMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &model.Commit{
		Hash:   "abc123",
		Author: "dev@example.com",
		Date:   time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
	}
	g.Expect(c.Hash).To(Equal("abc123"))
	g.Expect(c.Author).To(Equal("dev@example.com"))
	g.Expect(c.Date).To(Equal(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)))

	c.SetQuantity("lines-added", 42)
	v, ok := c.Quantity("lines-added")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(42)))
}

func TestCommit_SetAndGetMeasure(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &model.Commit{
		Hash:   "def456",
		Author: "dev@example.com",
		Date:   time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC),
	}
	g.Expect(c.Hash).To(Equal("def456"))
	g.Expect(c.Author).To(Equal("dev@example.com"))
	g.Expect(c.Date).To(Equal(time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC)))

	c.SetMeasure("churn-ratio", 0.75)
	v, ok := c.Measure("churn-ratio")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(BeNumerically("~", 0.75, 0.001))
}
