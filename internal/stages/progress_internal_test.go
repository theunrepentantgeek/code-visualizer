package stages

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// ---------------------------------------------------------------------------
// removeMetric
// ---------------------------------------------------------------------------

func TestRemoveMetric_RemovesFirstElement(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"a", "b", "c"}
	got := removeMetric(names, "a")

	g.Expect(got).To(Equal([]metric.Name{"b", "c"}))
}

func TestRemoveMetric_RemovesMiddleElement(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"a", "b", "c"}
	got := removeMetric(names, "b")

	g.Expect(got).To(Equal([]metric.Name{"a", "c"}))
}

func TestRemoveMetric_RemovesLastElement(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"a", "b", "c"}
	got := removeMetric(names, "c")

	g.Expect(got).To(Equal([]metric.Name{"a", "b"}))
}

func TestRemoveMetric_TargetAbsent_ReturnsSameSlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"a", "b", "c"}
	got := removeMetric(names, "z")

	g.Expect(got).To(Equal([]metric.Name{"a", "b", "c"}))
}

func TestRemoveMetric_EmptySlice_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	got := removeMetric(nil, "a")

	g.Expect(got).To(BeNil())
}

func TestRemoveMetric_SingleElement_Match_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"a"}
	got := removeMetric(names, "a")

	g.Expect(got).To(BeEmpty())
}
