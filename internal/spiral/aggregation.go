package spiral

import (
	"maps"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// commitCountMetric is the metric name for commit-count. In a spiral time bucket,
// each entry in b.Files represents one commit event, so the per-bucket value for
// this metric is naturally len(files) — not the sum of per-file lifetime totals.
const commitCountMetric metric.Name = "commit-count"

// AggregateBucketMetrics fills in SizeValue, FillValue, FillLabel, BorderValue,
// and BorderLabel for every bucket based on the files assigned to it. When
// sizeMetric is empty, SizeValue defaults to len(b.Files).
func AggregateBucketMetrics(
	buckets []TimeBucket,
	sizeMetric, fillMetric, borderMetric metric.Name,
) {
	for i := range buckets {
		aggregateBucket(&buckets[i], sizeMetric, fillMetric, borderMetric)
	}
}

func aggregateBucket(
	b *TimeBucket,
	sizeMetric, fillMetric, borderMetric metric.Name,
) {
	if sizeMetric != "" {
		b.SizeValue = bucketNumericValue(b.Files, sizeMetric)
	} else {
		b.SizeValue = float64(len(b.Files))
	}

	aggregateColourMetric(b.Files, fillMetric, &b.FillValue, &b.FillLabel)
	aggregateColourMetric(b.Files, borderMetric, &b.BorderValue, &b.BorderLabel)
}

func aggregateColourMetric(files []*model.File, m metric.Name, numVal *float64, catLabel *string) {
	if m == "" {
		return
	}

	d, ok := provider.GetDescriptor(m, metric.File)
	if !ok {
		return
	}

	if d.Kind == metric.Quantity || d.Kind == metric.Measure {
		*numVal = bucketNumericValue(files, m)
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
	if m == commitCountMetric {
		return float64(len(files))
	}

	return sumUniqueNumericMetric(files, m)
}

func sumUniqueNumericMetric(files []*model.File, m metric.Name) float64 {
	seen := map[*model.File]bool{}

	var total float64

	for _, f := range files {
		if seen[f] {
			continue
		}

		seen[f] = true

		if v, ok := f.Quantity(m); ok {
			total += float64(v)

			continue
		}

		if v, ok := f.Measure(m); ok {
			total += v
		}
	}

	return total
}

func modeCategory(files []*model.File, m metric.Name) string {
	counts := map[string]int{}

	for _, f := range files {
		if cat, ok := f.Classification(m); ok {
			counts[cat]++
		}
	}

	best := ""
	bestCount := 0

	for _, cat := range slices.Sorted(maps.Keys(counts)) {
		if counts[cat] > bestCount {
			best = cat
			bestCount = counts[cat]
		}
	}

	return best
}
