// Package metric defines file metrics (size, lines, type, git age/freshness/authors)
// and quantile-based bucketing for numeric values.
package metric

// Name identifies a metric. Provider packages define their own Name constants.
type Name string

// Kind describes the value type of a metric.
type Kind int

const (
	Quantity       Kind = iota // int values (file sizes, line counts)
	Measure                    // float64 values (percentages, rates)
	Classification             // string values (file type, category)
)

// Target classifies what a metric applies to.
type Target int

const (
	File      Target = iota // metric applies to individual files
	Directory               // metric applies to directories (aggregates)
)

// String returns the human-readable label for the target.
func (t Target) String() string {
	switch t {
	case File:
		return "file"
	case Directory:
		return "directory"
	default:
		return "unknown"
	}
}
