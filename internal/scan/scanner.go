package scan

import (
	"bufio"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

// Scan recursively scans the directory at path and returns a DirectoryNode tree.
// File symlinks are followed; directory symlinks are skipped.
// Permission-denied errors are logged and scanning continues.
// Returns an error if the directory contains no files.
func Scan(path string) (DirectoryNode, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return DirectoryNode{}, err
	}

	root, err := scanDir(absPath)
	if err != nil {
		return DirectoryNode{}, err
	}

	if countFiles(root) == 0 {
		return DirectoryNode{}, errors.New("no files found in directory")
	}

	return root, nil
}

func scanDir(dirPath string) (DirectoryNode, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return DirectoryNode{}, err
	}

	node := DirectoryNode{
		Path: dirPath,
		Name: filepath.Base(dirPath),
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())

		info, err := os.Stat(entryPath) // follows symlinks
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				slog.Warn("skipping file: permission denied", "path", entryPath)
				continue
			}
			slog.Warn("skipping file", "path", entryPath, "error", err)
			continue
		}

		if info.IsDir() {
			// Skip directory symlinks
			if isSymlink(entry) {
				slog.Debug("skipping directory symlink", "path", entryPath)
				continue
			}
			child, err := scanDir(entryPath)
			if err != nil {
				if errors.Is(err, fs.ErrPermission) {
					slog.Warn("skipping directory: permission denied", "path", entryPath)
					continue
				}
				return DirectoryNode{}, err
			}
			node.Dirs = append(node.Dirs, child)
		} else if info.Mode().IsRegular() || isSymlink(entry) {
			ext := strings.TrimPrefix(filepath.Ext(entry.Name()), ".")
			fileType := ext
			if fileType == "" {
				fileType = "no-extension"
			}
			node.Files = append(node.Files, FileNode{
				Path:      entryPath,
				Name:      entry.Name(),
				Extension: ext,
				Size:      info.Size(),
				FileType:  fileType,
			})
		}
	}

	return node, nil
}

func isSymlink(entry os.DirEntry) bool {
	return entry.Type()&os.ModeSymlink != 0
}

func countFiles(node DirectoryNode) int {
	count := len(node.Files)
	for _, d := range node.Dirs {
		count += countFiles(d)
	}
	return count
}

// PopulateLineCounts counts lines for all files in the tree.
// In non-git directories, all files are treated as text.
func PopulateLineCounts(node *DirectoryNode) {
	for i := range node.Files {
		f := &node.Files[i]
		if f.IsBinary {
			f.LineCount = 0
			continue
		}
		count, err := countLines(f.Path)
		if err != nil {
			slog.Warn("could not count lines", "path", f.Path, "error", err)
			continue
		}
		f.LineCount = count
	}
	for i := range node.Dirs {
		PopulateLineCounts(&node.Dirs[i])
	}
}

func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}
