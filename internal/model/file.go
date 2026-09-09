// Package model defines the tree data structure used by the metric framework.
package model

import (
	"io"
	"io/fs"

	"github.com/rotisserie/eris"
)

// File represents a single file in the scanned tree.
type File struct {
	MetricContainer
	Path         string
	RepoPath     string
	Name         string
	Extension    string
	IsBinary     bool
	Source       fs.FS
	Declarations []*Declaration
	Commits      []*Commit
}

// Open opens the file from its attached content source.
func (f *File) Open() (fs.File, error) {
	if f.Source == nil {
		return nil, eris.New("file has no content source")
	}

	file, err := f.Source.Open(f.Path)
	if err != nil {
		return nil, eris.Wrapf(err, "opening %s", f.Path)
	}

	return file, nil
}

// ReadAll reads the complete file from its attached content source.
func (f *File) ReadAll() ([]byte, error) {
	file, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, eris.Wrapf(err, "reading %s", f.Path)
	}

	return data, nil
}
