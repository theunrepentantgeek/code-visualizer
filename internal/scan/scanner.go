// Package scan provides recursive directory scanning with symlink handling
// and optional git metadata enrichment.
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

	"github.com/rotisserie/eris"
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
		return DirectoryNode{}, eris.Wrap(err, "failed to resolve absolute path")
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
		return DirectoryNode{}, eris.Wrapf(err, "failed to read directory %s", dirPath)
	}

	node := DirectoryNode{
		Path: dirPath,
		Name: filepath.Base(dirPath),
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())

		if err := processEntry(&node, entry, entryPath); err != nil {
			return DirectoryNode{}, err
		}
	}

	return node, nil
}

func processEntry(node *DirectoryNode, entry os.DirEntry, entryPath string) error {
	info, err := os.Stat(entryPath) // follows symlinks
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			slog.Warn("skipping file: permission denied", "path", entryPath)

			return nil
		}

		slog.Warn("skipping file", "path", entryPath, "error", err)

		return nil
	}

	if info.IsDir() {
		return processDir(node, entry, entryPath)
	}

	if info.Mode().IsRegular() || isSymlink(entry) {
		processFile(node, entry, info, entryPath)
	}

	return nil
}

func processDir(node *DirectoryNode, entry os.DirEntry, entryPath string) error {
	if isSymlink(entry) {
		slog.Debug("skipping directory symlink", "path", entryPath)

		return nil
	}

	child, err := scanDir(entryPath)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			slog.Warn("skipping directory: permission denied", "path", entryPath)

			return nil
		}

		return err
	}

	node.Dirs = append(node.Dirs, child)

	return nil
}

func processFile(node *DirectoryNode, entry os.DirEntry, info os.FileInfo, entryPath string) {
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

// EnrichWithGitMetadata populates git-derived fields (Age, Freshness, AuthorCount, IsBinary)
// for all files in the tree. rootPath is the absolute path of the scan root.
func EnrichWithGitMetadata(node *DirectoryNode, info *GitInfo, rootPath string) {
	for i := range node.Files {
		enrichFile(&node.Files[i], info, rootPath)
	}

	for i := range node.Dirs {
		EnrichWithGitMetadata(&node.Dirs[i], info, rootPath)
	}
}

func enrichFile(f *FileNode, info *GitInfo, rootPath string) {
	relPath, err := filepath.Rel(rootPath, f.Path)
	if err != nil {
		slog.Warn("could not compute relative path", "path", f.Path, "error", err)

		return
	}

	age, err := info.FileAge(relPath)
	if err != nil && !errors.Is(err, ErrUntracked) {
		slog.Debug("could not get file age", "path", relPath, "error", err)
	}

	f.Age = age

	freshness, err := info.FileFreshness(relPath)
	if err != nil && !errors.Is(err, ErrUntracked) {
		slog.Debug("could not get file freshness", "path", relPath, "error", err)
	}

	f.Freshness = freshness

	count, err := info.AuthorCount(relPath)
	if err != nil && !errors.Is(err, ErrUntracked) {
		slog.Debug("could not get author count", "path", relPath, "error", err)
	}

	f.AuthorCount = count

	f.IsBinary = info.IsBinary(relPath)
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
			if errors.Is(err, errBinaryFile) {
				f.IsBinary = true
				f.LineCount = 0

				continue
			}

			slog.Warn("could not count lines", "path", f.Path, "error", err)

			continue
		}

		f.LineCount = count
	}

	for i := range node.Dirs {
		PopulateLineCounts(&node.Dirs[i])
	}
}

var errBinaryFile = errors.New("file appears to be binary (line exceeds 64KB)")

// FilterBinaryFiles returns a copy of the directory tree with binary files removed.
// Directories that become empty after removal are also pruned.
// Each excluded file is logged at Debug level.
func FilterBinaryFiles(node DirectoryNode) DirectoryNode {
	result := DirectoryNode{
		Path: node.Path,
		Name: node.Name,
	}

	for _, f := range node.Files {
		if f.IsBinary {
			slog.Debug("excluding binary file", "path", f.Path)

			continue
		}

		result.Files = append(result.Files, f)
	}

	for _, d := range node.Dirs {
		filtered := FilterBinaryFiles(d)
		if len(filtered.Files) > 0 || len(filtered.Dirs) > 0 {
			result.Dirs = append(result.Dirs, filtered)
		}
	}

	return result
}

func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, eris.Wrapf(err, "failed to open file %s", path)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	count := 0
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, errBinaryFile
	}

	return count, nil
}
