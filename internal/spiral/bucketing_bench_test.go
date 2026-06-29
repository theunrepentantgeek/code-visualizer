package spiral

import (
	"testing"
	"time"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// BenchmarkAssignFilesToBuckets measures the performance of AssignFilesToBuckets
// on a workload representative of a mid-sized repository: 365 daily buckets
// (one year) with 1000 files each having 3 commits spread across the year.
func BenchmarkAssignFilesToBuckets(b *testing.B) {
	const (
		numFiles       = 1000
		commitsPerFile = 3
	)

	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := base.AddDate(1, 0, 0) // one year
	buckets := BuildTimeBuckets(Daily, base, end)
	span := end.Sub(base)

	files := make([]*model.File, numFiles)
	for i := range numFiles {
		files[i] = &model.File{Name: "file.go"}
	}

	history := make(map[*model.File][]stages.CommitRef, numFiles)
	for i, f := range files {
		refs := make([]stages.CommitRef, commitsPerFile)
		for j := range commitsPerFile {
			frac := time.Duration(float64(span) * float64(i*commitsPerFile+j) / float64(numFiles*commitsPerFile))
			refs[j] = stages.CommitRef{When: base.Add(frac)}
		}
		history[f] = refs
	}

	b.ResetTimer()

	for range b.N {
		// Re-slice bucket Files so each iteration starts clean without
		// re-allocating the full setup.
		for i := range buckets {
			buckets[i].Files = buckets[i].Files[:0]
		}
		AssignFilesToBuckets(buckets, history)
	}
}
