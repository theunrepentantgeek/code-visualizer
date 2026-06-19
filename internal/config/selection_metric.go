package config

import "slices"

// SelectionMetricRule maps files matching a glob pattern to a category string.
// Rules within a SelectionMetric are evaluated in order; the first match wins.
// A catch-all rule (Filename: "*") may be used as the final entry to assign
// a default category.
type SelectionMetricRule struct {
	Category string `yaml:"category" json:"category"`
	Filename string `yaml:"filename" json:"filename"` // doublestar glob matched against the file's relative path
}

// SelectionMetric defines a user-configured, filename-based classification metric.
//
// Each file is assigned the category of the first rule whose Filename glob pattern
// matches the file's relative path.  Files not matched by any rule have no value for
// this metric.
//
// Example (from config YAML):
//
//	selection-metrics:
//	  code-purpose:
//	    - category: test
//	      filename: "*_test.go"
//	    - category: source
//	      filename: "*"
type SelectionMetric struct {
	Name  string                `json:"name"          yaml:"name"`
	Rules []SelectionMetricRule `json:"rules"         yaml:"rules"`
}

// selectionMetricsRaw is the YAML/JSON on-disk format:
// a map from metric name → ordered rule list.  It matches the
// prototype YAML proposed in issue #402.
type selectionMetricsRaw map[string][]SelectionMetricRule

// toSlice converts the raw map into a name-sorted slice of SelectionMetric,
// populating the Name field from the map key.  Sorting ensures stable
// provider registration order across runs.
func (raw selectionMetricsRaw) toSlice() []SelectionMetric {
	if len(raw) == 0 {
		return nil
	}

	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	out := make([]SelectionMetric, len(keys))
	for i, k := range keys {
		out[i] = SelectionMetric{Name: k, Rules: raw[k]}
	}

	return out
}
