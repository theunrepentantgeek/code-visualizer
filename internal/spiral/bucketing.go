package spiral

import (
	"time"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// AssignFilesToBuckets distributes file-history records into pre-built time
// buckets. For each (*model.File, []stages.CommitRef) pair, each commit's
// timestamp is placed into the bucket whose half-open [Start, End) interval
// contains it. Files may appear in multiple buckets when they have multiple
// commits across time.
func AssignFilesToBuckets(
	buckets []TimeBucket,
	fileHistory map[*model.File][]stages.CommitRef,
) {
	for file, refs := range fileHistory {
		for _, ref := range refs {
			i := bucketIndexFor(buckets, ref.When)
			if i < 0 || i >= len(buckets) {
				continue
			}

			//nolint:gosec // Known safe index guarded by above bounds check
			buckets[i].Files = append(buckets[i].Files, file)
		}
	}
}

// bucketIndexFor returns the index of the bucket whose [Start, End) interval
// contains t. Returns -1 when t is out of range or the slice is empty.
//
// Buckets are normally uniform equal-duration intervals, so the index is
// computed in O(1) by dividing the elapsed time from the first bucket's start
// by the bucket duration. The computed bucket's [Start, End) interval is then
// verified to actually contain t; if it does not (because the buckets are
// non-uniform or non-consecutive), the function falls back to an O(N) linear
// scan so the correct index is still returned.
func bucketIndexFor(buckets []TimeBucket, t time.Time) int {
	if len(buckets) == 0 {
		return -1
	}

	dur := buckets[0].End.Sub(buckets[0].Start)
	if dur <= 0 {
		return linearBucketIndexFor(buckets, t)
	}

	delta := t.Sub(buckets[0].Start)
	if delta < 0 {
		return -1
	}

	i := int(delta / dur)
	if i >= 0 && i < len(buckets) && bucketContains(buckets[i], t) {
		return i
	}

	// The uniform-interval assumption did not hold; fall back to a linear scan.
	return linearBucketIndexFor(buckets, t)
}

// linearBucketIndexFor scans buckets sequentially and returns the index of the
// first bucket whose [Start, End) interval contains t, or -1 if none does.
func linearBucketIndexFor(buckets []TimeBucket, t time.Time) int {
	for i := range buckets {
		if bucketContains(buckets[i], t) {
			return i
		}
	}

	return -1
}

// bucketContains reports whether t falls within the bucket's half-open
// [Start, End) interval.
func bucketContains(bucket TimeBucket, t time.Time) bool {
	return !t.Before(bucket.Start) && t.Before(bucket.End)
}
