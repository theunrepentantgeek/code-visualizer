// Package metric defines file metrics (size, lines, type, git age/freshness/authors)
// and quantile-based bucketing for numeric values.
package metric

import "github.com/bevan/code-visualizer/internal/scan"

// MetricName identifies a metric used for sizing or colouring treemap rectangles.
type MetricName string

const (
	FileSize      MetricName = "file-size"
	FileLines     MetricName = "file-lines"
	FileType      MetricName = "file-type"
	FileAge       MetricName = "file-age"
	FileFreshness MetricName = "file-freshness"
	AuthorCount   MetricName = "author-count"
)

var validMetrics = map[MetricName]struct{}{
	FileSize:      {},
	FileLines:     {},
	FileType:      {},
	FileAge:       {},
	FileFreshness: {},
	AuthorCount:   {},
}

func (m MetricName) IsValid() bool {
	_, ok := validMetrics[m]
	return ok
}

func (m MetricName) IsNumeric() bool {
	return m != FileType && m.IsValid()
}

func (m MetricName) IsGitRequired() bool {
	return m == FileAge || m == FileFreshness || m == AuthorCount
}

// ExtractFileSize returns the file size in bytes as a float64.
func ExtractFileSize(node scan.FileNode) float64 {
	return float64(node.Size)
}

// ExtractFileLines returns the line count as a float64.
func ExtractFileLines(node scan.FileNode) float64 {
	return float64(node.LineCount)
}

// ExtractFileType returns the file type classification string.
func ExtractFileType(node scan.FileNode) string {
	return node.FileType
}
