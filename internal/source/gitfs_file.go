package source

import (
	"errors"
	"io"
	"io/fs"
	"time"
)

type gitFileInfo struct {
	name    string
	mode    fs.FileMode
	size    int64
	modTime time.Time
}

func (i gitFileInfo) Name() string       { return i.name }
func (i gitFileInfo) Size() int64        { return i.size }
func (i gitFileInfo) Mode() fs.FileMode  { return i.mode }
func (i gitFileInfo) ModTime() time.Time { return i.modTime }
func (i gitFileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i gitFileInfo) Sys() any           { return nil }

type gitDirEntry struct {
	info gitFileInfo
}

func (e gitDirEntry) Name() string               { return e.info.Name() }
func (e gitDirEntry) IsDir() bool                { return e.info.IsDir() }
func (e gitDirEntry) Type() fs.FileMode          { return e.info.Mode().Type() }
func (e gitDirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

type gitFile struct {
	reader io.ReadCloser
	info   gitFileInfo
}

func (f *gitFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *gitFile) Read(p []byte) (int, error) { return f.reader.Read(p) }
func (f *gitFile) Close() error               { return f.reader.Close() }

type gitDir struct {
	info    gitFileInfo
	entries []fs.DirEntry
	offset  int
	closed  bool
}

func (d *gitDir) Stat() (fs.FileInfo, error) { return d.info, nil }

func (d *gitDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.info.name, Err: errors.New("is a directory")}
}

func (d *gitDir) Close() error {
	d.closed = true

	return nil
}

func (d *gitDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.closed {
		return nil, &fs.PathError{Op: "readdir", Path: d.info.name, Err: fs.ErrClosed}
	}
	if d.offset >= len(d.entries) && n > 0 {
		return nil, io.EOF
	}

	end := len(d.entries)
	if n > 0 && d.offset+n < end {
		end = d.offset + n
	}

	entries := d.entries[d.offset:end]
	d.offset = end

	return entries, nil
}
