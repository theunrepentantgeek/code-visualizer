// Package metric defines file metrics (size, lines, type, git age/freshness/authors)
// and quantile-based bucketing for numeric values.
package metric

// MetricName identifies a metric used for sizing or colouring treemap rectangles.
type MetricName string

// Name identifies a metric. Provider packages define their own Name constants.
type Name = MetricName

// Kind describes the value type of a metric.
type Kind int

const (
	Quantity       Kind = iota // int values (file sizes, line counts)
	Measure                    // float64 values (percentages, rates)
	Classification             // string values (file type, category)
)

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
