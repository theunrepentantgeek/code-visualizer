package scan

import (
	"log/slog"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

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

	result.DirectFileCount = len(result.Files)
	allFileCount := result.DirectFileCount
	allDirCount := len(result.Dirs)

	for _, d := range result.Dirs {
		allFileCount += d.AllFileCount
		allDirCount += d.AllDirCount
	}

	result.AllFileCount = allFileCount
	result.AllDirCount = allDirCount

	return result
}
