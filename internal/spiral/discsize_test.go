package spiral

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// makeFiles creates n distinct dummy file pointers for testing.
func makeFiles(n int) []*model.File {
	files := make([]*model.File, n)
	for i := range n {
		files[i] = &model.File{Name: "file.go"}
	}

	return files
}

func TestApplyDiscSizes_LargestBucketGetsMaxDisc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	buckets := makeBuckets(5, Daily)

	nodes := make([]SpiralNode, 5)
	for i := range nodes {
		nodes[i].DiscRadius = defaultDiscRadius
	}

	// Give one bucket a clearly larger SizeValue.
	buckets[2].Files = makeFiles(10)
	buckets[0].Files = makeFiles(1)
	buckets[3].Files = makeFiles(5)

	for i := range buckets {
		buckets[i].SizeValue = float64(len(buckets[i].Files))
	}

	maxDisc := 20.0
	ApplyDiscSizes(nodes, buckets, maxDisc)

	// The largest bucket (index 2) should have radius == maxDisc.
	g.Expect(nodes[2].DiscRadius).To(BeNumerically("~", maxDisc, 1e-9), "largest bucket should get maxDisc radius")

	// Smaller active buckets should be between minDiscRadius and maxDisc.
	g.Expect(nodes[0].DiscRadius).To(BeNumerically(">=", minDiscRadius))
	g.Expect(nodes[0].DiscRadius).To(BeNumerically("<=", maxDisc))
	g.Expect(nodes[3].DiscRadius).To(BeNumerically(">=", minDiscRadius))
	g.Expect(nodes[3].DiscRadius).To(BeNumerically("<=", maxDisc))

	// Empty buckets should have zero radius.
	g.Expect(nodes[1].DiscRadius).To(Equal(0.0))
	g.Expect(nodes[4].DiscRadius).To(Equal(0.0))
}

func TestApplyDiscSizes_UsesReadableFloorWhenGeometrySupportsIt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	buckets := makeBuckets(3, Daily)
	maxDisc := 20.0

	nodes := make([]SpiralNode, 3)
	for i := range nodes {
		nodes[i].DiscRadius = defaultDiscRadius
		buckets[i].SizeValue = 0
		buckets[i].Files = makeFiles(1) // active but zero SizeValue
	}

	ApplyDiscSizes(nodes, buckets, maxDisc)

	for i := range nodes {
		g.Expect(nodes[i].DiscRadius).To(Equal(minDiscRadius))
		g.Expect(nodes[i].DiscRadius).To(BeNumerically("<=", maxDisc))
	}
}

func TestApplyDiscSizes_SmallerBucketIsSmallerThanLarger(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	buckets := makeBuckets(2, Daily)

	nodes := make([]SpiralNode, 2)
	for i := range nodes {
		nodes[i].DiscRadius = defaultDiscRadius
	}

	buckets[0].Files = makeFiles(1)
	buckets[0].SizeValue = 1.0
	buckets[1].Files = makeFiles(4)
	buckets[1].SizeValue = 4.0

	ApplyDiscSizes(nodes, buckets, 20.0)

	g.Expect(nodes[0].DiscRadius).To(BeNumerically("<", nodes[1].DiscRadius),
		"bucket with fewer commits should have smaller disc")
}

func TestApplyDiscSizes_UsesReadableFloorAndSquareRootScaling(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	buckets := makeBuckets(3, Daily)
	nodes := make([]SpiralNode, len(buckets))
	for i, size := range []float64{1, 4, 9} {
		buckets[i].Files = makeFiles(1)
		buckets[i].SizeValue = size
	}

	ApplyDiscSizes(nodes, buckets, 20)

	g.Expect(minDiscRadius).To(Equal(12.0))
	g.Expect(nodes[0].DiscRadius).To(BeNumerically(">", minDiscRadius))
	g.Expect(nodes[0].DiscRadius).To(BeNumerically("<", nodes[1].DiscRadius))
	g.Expect(nodes[1].DiscRadius).To(BeNumerically("<", nodes[2].DiscRadius))
	g.Expect(nodes[2].DiscRadius).To(BeNumerically("==", 20))
}

func TestApplyDiscSizes_DenseHourlyLayoutHonorsGeometryMaximum(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	buckets := BuildTimeBuckets(Hourly, start, start.Add(720*time.Hour))
	nodes := make([]SpiralNode, len(buckets))
	maxDisc := MaxDiscRadius(len(buckets), 1920, 1080, Hourly)

	g.Expect(maxDisc).To(BeNumerically(">", 0))
	g.Expect(maxDisc).To(BeNumerically("<", minDiscRadius))

	for i := range buckets {
		buckets[i].Files = makeFiles(1)
		buckets[i].SizeValue = float64(i%5 + 1)
	}

	ApplyDiscSizes(nodes, buckets, maxDisc)

	for i := range nodes {
		g.Expect(nodes[i].DiscRadius).To(BeNumerically("<=", maxDisc), "bucket %d", i)
	}
}
