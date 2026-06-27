package spiral

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// baseHour is a fixed reference point used across bucketing tests.
var baseHour = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// hourlyBuckets builds n consecutive hourly buckets starting at baseHour.
func hourlyBuckets(n int) []TimeBucket {
	return BuildTimeBuckets(Hourly, baseHour, baseHour.Add(time.Duration(n)*time.Hour))
}

// commitRef creates a CommitRef with only the When field set (Commit is nil,
// which is fine because AssignFilesToBuckets only reads When).
func commitRef(t time.Time) stages.CommitRef {
	return stages.CommitRef{When: t}
}

// TestAssignFilesToBuckets_FileWithCommitInBucket_IsAssigned verifies that a
// commit whose timestamp falls within a bucket's [Start, End) interval causes
// the file to appear in that bucket.
func TestAssignFilesToBuckets_FileWithCommitInBucket_IsAssigned(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &model.File{Name: "foo.go"}
	buckets := hourlyBuckets(3) // [0,1h), [1h,2h), [2h,3h)

	// commit mid-way through bucket 1
	history := map[*model.File][]stages.CommitRef{
		f: {commitRef(baseHour.Add(90 * time.Minute))},
	}

	AssignFilesToBuckets(buckets, history)

	g.Expect(buckets[0].Files).To(BeEmpty())
	g.Expect(buckets[1].Files).To(ConsistOf(f))
	g.Expect(buckets[2].Files).To(BeEmpty())
}

// TestAssignFilesToBuckets_CommitBeforeBuckets_IsIgnored verifies that commits
// with timestamps before the first bucket start are silently skipped.
func TestAssignFilesToBuckets_CommitBeforeBuckets_IsIgnored(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &model.File{Name: "before.go"}
	buckets := hourlyBuckets(3)

	history := map[*model.File][]stages.CommitRef{
		f: {commitRef(baseHour.Add(-time.Minute))},
	}

	AssignFilesToBuckets(buckets, history)

	for i := range buckets {
		g.Expect(buckets[i].Files).To(BeEmpty(), "bucket %d should be empty", i)
	}
}

// TestAssignFilesToBuckets_CommitAfterBuckets_IsIgnored verifies that commits
// with timestamps at or after the last bucket's End are silently skipped.
func TestAssignFilesToBuckets_CommitAfterBuckets_IsIgnored(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &model.File{Name: "after.go"}
	buckets := hourlyBuckets(3) // ends at baseHour+3h

	history := map[*model.File][]stages.CommitRef{
		f: {commitRef(baseHour.Add(4 * time.Hour))},
	}

	AssignFilesToBuckets(buckets, history)

	for i := range buckets {
		g.Expect(buckets[i].Files).To(BeEmpty(), "bucket %d should be empty", i)
	}
}

// TestAssignFilesToBuckets_HalfOpenInterval_StartIsInclusive verifies that a
// commit whose timestamp equals a bucket's Start falls in that bucket (not the
// previous one), confirming the [Start, End) half-open semantics.
func TestAssignFilesToBuckets_HalfOpenInterval_StartIsInclusive(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &model.File{Name: "boundary.go"}
	buckets := hourlyBuckets(3)

	// commit exactly at start of bucket 1
	history := map[*model.File][]stages.CommitRef{
		f: {commitRef(baseHour.Add(time.Hour))},
	}

	AssignFilesToBuckets(buckets, history)

	g.Expect(buckets[0].Files).To(BeEmpty(), "commit at bucket 1 Start should NOT be in bucket 0")
	g.Expect(buckets[1].Files).To(ConsistOf(f), "commit at bucket 1 Start should be in bucket 1")
	g.Expect(buckets[2].Files).To(BeEmpty())
}

// TestAssignFilesToBuckets_HalfOpenInterval_EndIsExclusive verifies that a
// commit whose timestamp equals a bucket's End falls in the next bucket (not
// the current one), confirming the [Start, End) half-open semantics.
func TestAssignFilesToBuckets_HalfOpenInterval_EndIsExclusive(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &model.File{Name: "boundary.go"}
	buckets := hourlyBuckets(3)

	// commit at exact End of bucket 0 (= Start of bucket 1)
	history := map[*model.File][]stages.CommitRef{
		f: {commitRef(buckets[0].End)},
	}

	AssignFilesToBuckets(buckets, history)

	g.Expect(buckets[0].Files).To(BeEmpty(), "commit at bucket 0 End should NOT be in bucket 0")
	g.Expect(buckets[1].Files).To(ConsistOf(f), "commit at bucket 0 End should be in bucket 1")
}

// TestAssignFilesToBuckets_FileAppearsInMultipleBuckets verifies that a file
// with commits spread across different buckets is added to each relevant bucket.
func TestAssignFilesToBuckets_FileAppearsInMultipleBuckets(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &model.File{Name: "multi.go"}
	buckets := hourlyBuckets(3)

	history := map[*model.File][]stages.CommitRef{
		f: {
			commitRef(baseHour.Add(30 * time.Minute)),  // bucket 0
			commitRef(baseHour.Add(150 * time.Minute)), // bucket 2
		},
	}

	AssignFilesToBuckets(buckets, history)

	g.Expect(buckets[0].Files).To(ConsistOf(f))
	g.Expect(buckets[1].Files).To(BeEmpty())
	g.Expect(buckets[2].Files).To(ConsistOf(f))
}

// TestAssignFilesToBuckets_MultipleFiles verifies that independent files with
// commits in distinct buckets each appear only in the correct bucket.
func TestAssignFilesToBuckets_MultipleFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f0 := &model.File{Name: "a.go"}
	f1 := &model.File{Name: "b.go"}
	f2 := &model.File{Name: "c.go"}
	buckets := hourlyBuckets(3)

	history := map[*model.File][]stages.CommitRef{
		f0: {commitRef(baseHour.Add(10 * time.Minute))},
		f1: {commitRef(baseHour.Add(70 * time.Minute))},
		f2: {commitRef(baseHour.Add(130 * time.Minute))},
	}

	AssignFilesToBuckets(buckets, history)

	g.Expect(buckets[0].Files).To(ConsistOf(f0))
	g.Expect(buckets[1].Files).To(ConsistOf(f1))
	g.Expect(buckets[2].Files).To(ConsistOf(f2))
}

// TestAssignFilesToBuckets_EmptyHistory verifies that passing an empty file
// history leaves all bucket Files slices empty (no panics, no side effects).
func TestAssignFilesToBuckets_EmptyHistory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := hourlyBuckets(3)

	AssignFilesToBuckets(buckets, map[*model.File][]stages.CommitRef{})

	for i := range buckets {
		g.Expect(buckets[i].Files).To(BeEmpty(), "bucket %d should be empty", i)
	}
}

// TestAssignFilesToBuckets_EmptyBuckets verifies that passing an empty bucket
// slice with a non-empty history does not panic.
func TestAssignFilesToBuckets_EmptyBuckets(t *testing.T) {
	t.Parallel()

	f := &model.File{Name: "foo.go"}
	history := map[*model.File][]stages.CommitRef{
		f: {commitRef(baseHour)},
	}

	// Should not panic; all commits are silently skipped because bucketIndexFor
	// returns -1 when the bucket slice is empty.
	AssignFilesToBuckets([]TimeBucket{}, history)
}
