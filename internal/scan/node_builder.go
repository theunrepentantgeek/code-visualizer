package scan

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

type binaryProbe func(path string) (bool, error)

type nodeBuilder struct {
	probe binaryProbe
}

func newNodeBuilder(probe binaryProbe) nodeBuilder {
	if probe == nil {
		probe = IsBinaryFile
	}

	return nodeBuilder{probe: probe}
}

func (b nodeBuilder) processFile(node *model.Directory, entry os.DirEntry, info os.FileInfo, entryPath string) {
	ext := strings.TrimPrefix(filepath.Ext(entry.Name()), ".")

	fileType := ext
	if fileType == "" {
		fileType = "no-extension"
	}

	binary, err := b.probe(entryPath)
	if err != nil {
		slog.Warn("binary probe failed, assuming text", "path", entryPath, "error", err)
	}

	file := &model.File{
		Path:      entryPath,
		Name:      entry.Name(),
		Extension: ext,
		IsBinary:  binary,
	}

	file.SetQuantity(filesystem.FileSize, info.Size())
	file.SetClassification(filesystem.FileType, fileType)

	node.Files = append(node.Files, file)
}

func hasFiles(node *model.Directory) bool {
	if len(node.Files) > 0 {
		return true
	}

	return slices.ContainsFunc(node.Dirs, hasFiles)
}

func FilterBinaryFiles(node *model.Directory) *model.Directory {
	result := &model.Directory{
		Path:  node.Path,
		Name:  node.Name,
		Files: make([]*model.File, 0, len(node.Files)),
		Dirs:  make([]*model.Directory, 0, len(node.Dirs)),
	}

	for _, file := range node.Files {
		if file.IsBinary {
			slog.Debug("excluding binary file", "path", file.Path)

			continue
		}

		result.Files = append(result.Files, file)
	}

	for _, dir := range node.Dirs {
		filtered := FilterBinaryFiles(dir)
		if len(filtered.Files) > 0 || len(filtered.Dirs) > 0 {
			result.Dirs = append(result.Dirs, filtered)
		}
	}

	return result
}
