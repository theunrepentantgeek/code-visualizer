package metric

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
