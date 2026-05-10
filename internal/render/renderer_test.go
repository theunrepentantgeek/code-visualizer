package render

import (
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
)

func makeFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}
