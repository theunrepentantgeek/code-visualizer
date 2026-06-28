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

func bucketIndexFor(buckets []TimeBucket, t time.Time) int {
	for i := range buckets {
		if !t.Before(buckets[i].Start) && t.Before(buckets[i].End) {
			return i
		}
	}

	return -1
}
