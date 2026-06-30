package model

// Directory represents a directory in the scanned tree.
type Directory struct {
	MetricContainer
	Path  string
	Name  string
	Files []*File
	Dirs  []*Directory

	// DirectFileCount is the number of files directly in this directory (not in subdirectories).
	// Populated during the file scan; zero if the directory was constructed manually.
	DirectFileCount int

	// AllFileCount is the total number of files in this directory and all descendants.
	// Populated during the file scan; zero if the directory was constructed manually.
	AllFileCount int
}
