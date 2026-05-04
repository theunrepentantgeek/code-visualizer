// Package model defines the tree data structure used by the metric framework.
package model

// File represents a single file in the scanned tree.
type File struct {
	MetricBag
	Path      string
	Name      string
	Extension string
	IsBinary  bool
}
