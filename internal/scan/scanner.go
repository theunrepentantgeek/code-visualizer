package scan

import "time"

// FileNode represents a single file discovered during directory scanning.
type FileNode struct {
	Path        string
	Name        string
	Extension   string
	Size        int64
	LineCount   int
	FileType    string
	Age         *time.Duration
	Freshness   *time.Duration
	AuthorCount *int
	IsBinary    bool
}

// DirectoryNode represents a directory in the scanned tree.
type DirectoryNode struct {
	Path  string
	Name  string
	Files []FileNode
	Dirs  []DirectoryNode
}
