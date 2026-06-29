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
// Buckets are uniform equal-duration intervals so the index is computed in
// O(1) by dividing the elapsed time from the first bucket's start by the
// bucket duration, instead of scanning the slice linearly.
func bucketIndexFor(buckets []TimeBucket, t time.Time) int {
	if len(buckets) == 0 {
		return -1
	}

	dur := buckets[0].End.Sub(buckets[0].Start)
	if dur <= 0 {
		return -1
	}

	delta := t.Sub(buckets[0].Start)
	if delta < 0 {
		return -1
	}

	i := int(delta / dur)
	if i >= len(buckets) {
		return -1
	}

	return i
}
