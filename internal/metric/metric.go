// Package metric defines file metrics (size, lines, type, git age/freshness/authors)
// and quantile-based bucketing for numeric values.
package metric

// Name identifies a metric. Provider packages define their own Name constants.
type Name string

// MetricName is a deprecated alias for Name.
type MetricName = Name

// Kind describes the value type of a metric.
type Kind int

const (
	Quantity       Kind = iota // int values (file sizes, line counts)
	Measure                    // float64 values (percentages, rates)
	Classification             // string values (file type, category)
)
