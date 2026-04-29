package spiral

import (
	"time"

	"github.com/bevan/code-visualizer/internal/model"
)

// Resolution controls the time granularity of the spiral.
type Resolution int

const (
	// Hourly uses 24 spots per lap (one lap = one day).
	Hourly Resolution = iota
	// Daily uses 28 spots per lap (one lap = four weeks).
	Daily
)

// SpotsPerLap returns the number of time buckets in one full revolution.
func (r Resolution) SpotsPerLap() int {
	switch r {
	case Daily:
		return 28
	default:
		return 24
	}
}

// bucketDuration returns the duration of a single time bucket at this resolution.
func (r Resolution) bucketDuration() time.Duration {
	switch r {
	case Daily:
		return 24 * time.Hour
	default:
		return time.Hour
	}
}

// TimeBucket represents a single time interval on the spiral.
type TimeBucket struct {
	Start time.Time     // inclusive
	End   time.Time     // exclusive
	Files []*model.File // files whose activity falls in this bucket

	// Aggregated metric values (populated after bucket assignment by the CLI layer).
	SizeValue   float64
	FillValue   float64
	FillLabel   string
	BorderValue float64
	BorderLabel string
}

// BuildTimeBuckets creates consecutive time buckets at the given resolution
// spanning from startTime to endTime. Each bucket covers one unit of resolution
// (one hour for Hourly, one day for Daily). The last bucket may extend slightly
// past endTime to complete its unit.
func BuildTimeBuckets(
	resolution Resolution,
	startTime time.Time,
	endTime time.Time,
) []TimeBucket {
	if !endTime.After(startTime) {
		return []TimeBucket{}
	}

	dur := resolution.bucketDuration()
	start := truncateToResolution(startTime, resolution)

	buckets := make([]TimeBucket, 0)

	for t := start; t.Before(endTime); t = t.Add(dur) {
		buckets = append(buckets, TimeBucket{
			Start: t,
			End:   t.Add(dur),
		})
	}

	return buckets
}

// truncateToResolution rounds t down to the start of its resolution unit.
func truncateToResolution(t time.Time, r Resolution) time.Time {
	switch r {
	case Daily:
		y, m, d := t.Date()

		return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	default:
		return t.Truncate(time.Hour)
	}
}
