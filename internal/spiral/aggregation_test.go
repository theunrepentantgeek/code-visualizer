package spiral

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const sizeMetric metric.Name = "file-size"

// makeFileWithSize creates a test file with a quantity metric set.
func makeFileWithSize(size int64) *model.File {
	f := &model.File{Name: "test.go"}
	f.SetQuantity(sizeMetric, size)

	return f
}

// TestBucketNumericValue_CommitCountUsesLen verifies that commit-count returns
// len(files) — the count of commit events — rather than summing per-file lifetime
// values. This is the core of the bug fix for issue #253.
func TestBucketNumericValue_CommitCountUsesLen(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Same file appearing 3 times (as if it was committed 3 times in this bucket).
	f := &model.File{Name: "busy.go"}
	files := []*model.File{f, f, f}

	result := bucketNumericValue(files, commitCountMetric)
	g.Expect(result).To(Equal(3.0))
}

// TestBucketNumericValue_CommitCountDistinctFiles confirms that distinct files
// each counting as one commit event also produce the correct count.
func TestBucketNumericValue_CommitCountDistinctFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []*model.File{
		{Name: "a.go"},
		{Name: "b.go"},
		{Name: "c.go"},
	}

	result := bucketNumericValue(files, commitCountMetric)
	g.Expect(result).To(Equal(3.0))
}

// TestBucketNumericValue_CommitCountEmpty confirms zero for an empty bucket.
func TestBucketNumericValue_CommitCountEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := bucketNumericValue(nil, commitCountMetric)
	g.Expect(result).To(Equal(0.0))
}

// TestBucketNumericValue_OtherMetricDeduplicates verifies that non-commit-count
// metrics are summed across unique files (duplicates are ignored).
func TestBucketNumericValue_OtherMetricDeduplicates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	f1 := makeFileWithSize(100)
	f2 := makeFileWithSize(50)

	// f1 appears twice (two commits), f2 appears once. The file-size metric
	// should only count f1 once (deduplicated).
	files := []*model.File{f1, f1, f2}

	result := bucketNumericValue(files, sizeMetric)
	g.Expect(result).To(Equal(150.0)) // f1(100) + f2(50), f1 not double-counted
}

// TestAggregateBucketMetrics_CommitCountSize verifies that specifying
// "commit-count" as the size metric produces the count of commit events per
// bucket, not the sum of per-file lifetime commit totals.
func TestAggregateBucketMetrics_CommitCountSize(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	f := &model.File{Name: "active.go"}

	buckets := []TimeBucket{
		{Files: []*model.File{f, f, f}},  // 3 commit events
		{Files: []*model.File{f}},         // 1 commit event
		{Files: []*model.File{}},          // empty bucket
	}

	AggregateBucketMetrics(buckets, commitCountMetric, "", "")

	g.Expect(buckets[0].SizeValue).To(Equal(3.0))
	g.Expect(buckets[1].SizeValue).To(Equal(1.0))
	g.Expect(buckets[2].SizeValue).To(Equal(0.0))
}
