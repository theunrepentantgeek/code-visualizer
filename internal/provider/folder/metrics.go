// Package folder provides metric providers for folder-level aggregate metrics.
package folder

import (
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
)

// Metric name constants for folder-level metrics.
const (
	FolderAuthorCount metric.Name = "folder-author-count"
	FolderAge         metric.Name = "folder-age"
	FolderFreshness   metric.Name = "folder-freshness"
	TotalFolderLines  metric.Name = "total-folder-lines"
	TotalFolderSize   metric.Name = "total-folder-size"
	MeanFileAge       metric.Name = "mean-file-age"
	MeanFileFreshness metric.Name = "mean-file-freshness"
	MeanFileLines     metric.Name = "mean-file-lines"
	MeanFileSize      metric.Name = "mean-file-size"
)

// sumCount accumulates a running total for computing arithmetic means.
type sumCount struct {
	sum   float64
	count int64
}

// maxQuantityFromFiles returns the maximum quantity value from the direct files in d.
func maxQuantityFromFiles(d *model.Directory, name metric.Name) (int64, bool) {
	var result int64

	found := false

	for _, f := range d.Files {
		if v, ok := f.Quantity(name); ok {
			if !found || v > result {
				result = v
				found = true
			}
		}
	}

	return result, found
}

// maxQuantityFromDirs returns the maximum quantity value from the direct subdirectories of d.
func maxQuantityFromDirs(d *model.Directory, name metric.Name) (int64, bool) {
	var result int64

	found := false

	for _, sub := range d.Dirs {
		if v, ok := sub.Quantity(name); ok {
			if !found || v > result {
				result = v
				found = true
			}
		}
	}

	return result, found
}

// minQuantityFromFiles returns the minimum quantity value from the direct files in d.
func minQuantityFromFiles(d *model.Directory, name metric.Name) (int64, bool) {
	var result int64

	found := false

	for _, f := range d.Files {
		if v, ok := f.Quantity(name); ok {
			if !found || v < result {
				result = v
				found = true
			}
		}
	}

	return result, found
}

// minQuantityFromDirs returns the minimum quantity value from the direct subdirectories of d.
func minQuantityFromDirs(d *model.Directory, name metric.Name) (int64, bool) {
	var result int64

	found := false

	for _, sub := range d.Dirs {
		if v, ok := sub.Quantity(name); ok {
			if !found || v < result {
				result = v
				found = true
			}
		}
	}

	return result, found
}

// sumQuantityFromFiles returns the sum of quantity values from the direct files in d.
func sumQuantityFromFiles(d *model.Directory, name metric.Name) (int64, bool) {
	var total int64

	found := false

	for _, f := range d.Files {
		if v, ok := f.Quantity(name); ok {
			total += v
			found = true
		}
	}

	return total, found
}

// sumQuantityFromDirs returns the sum of quantity values from direct subdirectories of d.
func sumQuantityFromDirs(d *model.Directory, name metric.Name) (int64, bool) {
	var total int64

	found := false

	for _, sub := range d.Dirs {
		if v, ok := sub.Quantity(name); ok {
			total += v
			found = true
		}
	}

	return total, found
}

// addSumCountFromFiles adds the quantity values from direct files in d to sc.
func addSumCountFromFiles(sc *sumCount, d *model.Directory, name metric.Name) {
	for _, f := range d.Files {
		if v, ok := f.Quantity(name); ok {
			sc.sum += float64(v)
			sc.count++
		}
	}
}

// addPositiveSumCountFromFiles adds positive quantity values from direct files in d to sc.
// Files where the value is zero are skipped (used to exclude binary files from line-count means).
func addPositiveSumCountFromFiles(sc *sumCount, d *model.Directory, name metric.Name) {
	for _, f := range d.Files {
		if v, ok := f.Quantity(name); ok && v > 0 {
			sc.sum += float64(v)
			sc.count++
		}
	}
}

// addSumCountFromDirs merges accumulated stats from direct subdirectories of d into sc.
func addSumCountFromDirs(sc *sumCount, d *model.Directory, stats map[string]*sumCount) {
	for _, sub := range d.Dirs {
		if ss, ok := stats[sub.Path]; ok {
			sc.sum += ss.sum
			sc.count += ss.count
		}
	}
}
