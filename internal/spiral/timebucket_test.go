package spiral

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestBuildTimeBucketsHourly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Hourly, start, end)
	g.Expect(buckets).To(HaveLen(3))
	g.Expect(buckets[0].Start).To(Equal(start))
	g.Expect(buckets[0].End).To(Equal(start.Add(time.Hour)))
	g.Expect(buckets[2].End).To(Equal(end))
}

func TestBuildTimeBucketsDaily(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Daily, start, end)
	g.Expect(buckets).To(HaveLen(3))
	g.Expect(buckets[0].Start).To(Equal(start))
	g.Expect(buckets[2].End).To(Equal(end))
}

func TestBuildTimeBucketsEndBeforeStart(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	start := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Hourly, start, end)
	g.Expect(buckets).To(BeNil())
}

func TestBuildTimeBucketsEqualStartEnd(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Daily, t0, t0)
	g.Expect(buckets).To(BeNil())
}

func TestBuildTimeBucketsTruncatesStart(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Start mid-hour — should truncate to hour boundary.
	start := time.Date(2026, 1, 1, 2, 30, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 5, 0, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Hourly, start, end)
	g.Expect(buckets[0].Start).To(Equal(
		time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),
	))
}

func TestBuildTimeBucketsDailyTruncatesStart(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Start mid-day — should truncate to midnight.
	start := time.Date(2026, 1, 1, 14, 30, 0, 0, time.UTC)
	end := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Daily, start, end)
	g.Expect(buckets[0].Start).To(Equal(
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	))
}

func TestBuildTimeBucketsPartialEnd(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 2, 30, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Hourly, start, end)
	// Should include 3 buckets: 0-1, 1-2, 2-3 (last one extends past end).
	g.Expect(buckets).To(HaveLen(3))
}

func TestBuildTimeBucketsFilesFieldInitialized(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Hourly, start, end)
	g.Expect(buckets).To(HaveLen(1))
	g.Expect(buckets[0].Files).To(BeNil())
}

func TestResolutionSpotsPerLap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(Hourly.SpotsPerLap()).To(Equal(24))
	g.Expect(Daily.SpotsPerLap()).To(Equal(28))

	// Unknown resolution falls back to 24.
	unknown := Resolution(99)
	g.Expect(unknown.SpotsPerLap()).To(Equal(24))
}

// --- Gap tests added by Lambert (Phase 4, issue #127) ---

func TestBuildTimeBucketsHourly24Hours(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Hourly, start, end)
	g.Expect(buckets).To(HaveLen(24), "24 hours should produce 24 hourly buckets")
	g.Expect(buckets[0].Start).To(Equal(start))
	g.Expect(buckets[23].End).To(Equal(end))
}

func TestBuildTimeBucketsDaily28Days(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 29, 0, 0, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Daily, start, end)
	g.Expect(buckets).To(HaveLen(28), "28 days should produce 28 daily buckets")
	g.Expect(buckets[0].Start).To(Equal(start))
	g.Expect(buckets[27].End).To(Equal(end))
}

func TestBuildTimeBucketsHalfOpenIntervals(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 5, 0, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Hourly, start, end)
	g.Expect(buckets).To(HaveLen(5))

	for i, b := range buckets {
		// Each bucket Start should equal previous bucket End (half-open intervals).
		g.Expect(b.End).To(Equal(b.Start.Add(time.Hour)),
			"bucket %d should span exactly 1 hour", i)

		if i > 0 {
			g.Expect(b.Start).To(Equal(buckets[i-1].End),
				"bucket %d Start should equal bucket %d End (contiguous)", i, i-1)
		}
	}

	// Exact boundary: 2:00:00 should be the Start of bucket 2, not End of bucket 1.
	g.Expect(buckets[2].Start).To(Equal(
		time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC)),
		"bucket 2 should start at exactly 2:00")
}

func TestBuildTimeBucketsMidnightBoundary(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Span the midnight boundary.
	start := time.Date(2026, 1, 1, 22, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 2, 2, 0, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Hourly, start, end)
	g.Expect(buckets).To(HaveLen(4)) // 22, 23, 0, 1

	midnight := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	g.Expect(buckets[1].End).To(Equal(midnight),
		"bucket spanning 23:00 should end at midnight")
	g.Expect(buckets[2].Start).To(Equal(midnight),
		"bucket starting at midnight should be the next day's first hour")
}

func TestBuildTimeBucketsDailyMidnightBoundary(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// A midnight-aligned start should produce a clean day boundary.
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Daily, start, end)
	g.Expect(buckets).To(HaveLen(2))

	day2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	g.Expect(buckets[0].End).To(Equal(day2),
		"first daily bucket should end at midnight of day 2")
	g.Expect(buckets[1].Start).To(Equal(day2),
		"second daily bucket should start at midnight of day 2")
}

func TestBuildTimeBucketsSingleUnit(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	start := time.Date(2026, 1, 1, 5, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 5, 30, 0, 0, time.UTC) // Half an hour

	buckets := BuildTimeBuckets(Hourly, start, end)
	g.Expect(buckets).To(HaveLen(1), "a sub-hour range should produce 1 bucket")
	g.Expect(buckets[0].Start).To(Equal(start))
	g.Expect(buckets[0].End).To(Equal(start.Add(time.Hour)))
}

func TestBuildTimeBucketsContiguous(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Hourly, start, end)

	for i := 1; i < len(buckets); i++ {
		g.Expect(buckets[i].Start).To(Equal(buckets[i-1].End),
			"buckets %d and %d should be contiguous with no gap", i-1, i)
	}
}

func TestBuildTimeBucketsDailyContiguous(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC) // 7 days

	buckets := BuildTimeBuckets(Daily, start, end)
	g.Expect(buckets).To(HaveLen(7))

	for i := 1; i < len(buckets); i++ {
		g.Expect(buckets[i].Start).To(Equal(buckets[i-1].End),
			"daily buckets %d and %d should be contiguous", i-1, i)
	}
}

func TestBuildTimeBucketsDailyEachSpans24Hours(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)

	buckets := BuildTimeBuckets(Daily, start, end)
	g.Expect(buckets).To(HaveLen(3))

	for i, b := range buckets {
		dur := b.End.Sub(b.Start)
		g.Expect(dur).To(Equal(24*time.Hour),
			"daily bucket %d should span exactly 24 hours", i)
	}
}
