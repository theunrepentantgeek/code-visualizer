package model

// Directory represents a directory in the scanned tree.
type Directory struct {
	MetricBag
	Path  string
	Name  string
	Files []*File
	Dirs  []*Directory
}
