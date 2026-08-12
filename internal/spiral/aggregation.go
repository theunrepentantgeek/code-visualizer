package spiral

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// commitCountMetric is the metric name for commit-count. In a spiral time bucket,
// each entry in b.Files represents one commit event, so the per-bucket value for
// this metric is naturally len(files) — not the sum of per-file lifetime totals.
const commitCountMetric metric.Name = "commit-count"

// AggregateBucketMetrics fills in metric values for every bucket based on the
// files assigned to it. When sizeMetric is empty, SizeValue defaults to
// len(b.Files).
func AggregateBucketMetrics(
	buckets []TimeBucket,
	requested stages.RequestedMetrics,
	sizeMetric, fillMetric, borderMetric, surfaceMetric metric.Name,
) {
	for i := range buckets {
		aggregateBucket(&buckets[i], requested, sizeMetric, fillMetric, borderMetric, surfaceMetric)
	}
}

func aggregateBucket(
	b *TimeBucket,
	requested stages.RequestedMetrics,
	sizeMetric, fillMetric, borderMetric, surfaceMetric metric.Name,
) {
	if sizeMetric != "" {
		b.SizeValue, b.SizeValueAvailable = bucketNumericMetricValue(b.Files, sizeMetric)
	} else {
		b.SizeValue = float64(len(b.Files))
		b.SizeValueAvailable = true
	}

	aggregateColourMetric(b.Files, fillMetric, requested, &b.FillValue, &b.FillValueAvailable, &b.FillLabel)
	aggregateColourMetric(b.Files, borderMetric, requested, &b.BorderValue, &b.BorderValueAvailable, &b.BorderLabel)

	if surfaceMetric != "" {
		b.SurfaceValue, b.SurfaceValueAvailable = bucketNumericMetricValue(b.Files, surfaceMetric)
	}
}

func aggregateColourMetric(
	files []*model.File,
	m metric.Name,
	requested stages.RequestedMetrics,
	numVal *float64,
	numAvailable *bool,
	catLabel *string,
) {
	if m == "" {
		return
	}

	d, ok := requested.DescriptorFor(m)
	if !ok {
		return
	}

	if d.Kind == metric.Quantity || d.Kind == metric.Measure {
		*numVal, *numAvailable = bucketNumericMetricValue(files, m)
	} else {
		*catLabel = modeCategory(files, m)
	}
}

// bucketNumericValue returns the aggregated numeric value for metric m across
// the files in a time bucket. For commit-count, the natural per-bucket value
// is the number of commit events (len(files)) because each entry in files
// represents one commit, so summing per-file lifetime totals would be wrong.
// For all other numeric metrics, values are summed across unique files.
func bucketNumericValue(files []*model.File, m metric.Name) float64 {
	value, _ := bucketNumericMetricValue(files, m)

	return value
}

func bucketNumericMetricValue(files []*model.File, m metric.Name) (float64, bool) {
	if m == commitCountMetric {
		return float64(len(files)), true
	}

	return sumUniqueNumericMetric(files, m)
}

func sumUniqueNumericMetric(files []*model.File, m metric.Name) (float64, bool) {
	seen := map[*model.File]bool{}

	var (
		total     float64
		available bool
	)

	for _, f := range files {
		if seen[f] {
			continue
		}

		seen[f] = true

		if v, ok := f.Quantity(m); ok {
			total += float64(v)
			available = true

			continue
		}

		if v, ok := f.Measure(m); ok {
			total += v
			available = true
		}
	}

	return total, available
}

func modeCategory(files []*model.File, m metric.Name) string {
	counts := make(map[string]int, len(files))

	best := ""
	maxCount := 0

	for _, f := range files {
		cat, ok := f.Classification(m)
		if !ok {
			continue
		}

		counts[cat]++
		count := counts[cat]

		if count > maxCount || (count == maxCount && cat < best) {
			best = cat
			maxCount = count
		}
	}

	return best
}
