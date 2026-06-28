package goldentest

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

// synthFile builds a model.File with the file-level base metrics every
// visualization may read. lines must be distinct across siblings to keep the
// (unstable) radius sort deterministic.
func synthFile(path, name, ext, fileType string, lines, size int64) *model.File {
	f := &model.File{
		Path:      path,
		Name:      name,
		Extension: ext,
	}
	f.SetQuantity(filesystem.FileLines, lines)
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, fileType)

	return f
}

// buildVizModel returns a fixed, deterministic directory tree for the
// visualization golden tests. Three levels deep with a spread of file types and
// distinct sizes so layouts and colour scales are non-trivial.
func buildVizModel() *model.Directory {
	return &model.Directory{
		Path: rootDirName,
		Name: rootDirName,
		Files: []*model.File{
			synthFile("root/readme.md", "readme.md", "md", "Markdown", 40, 800),
			synthFile("root/main.go", "main.go", "go", "Go", 120, 2400),
		},
		Dirs: []*model.Directory{
			{
				Path: "root/src",
				Name: "src",
				Files: []*model.File{
					synthFile("root/src/app.go", "app.go", "go", "Go", 210, 4100),
					synthFile("root/src/util.go", "util.go", "go", "Go", 75, 1500),
					synthFile("root/src/styles.css", "styles.css", "css", "CSS", 33, 660),
				},
				Dirs: []*model.Directory{
					{
						Path: "root/src/deep",
						Name: "deep",
						Files: []*model.File{
							synthFile("root/src/deep/big.go", "big.go", "go", "Go", 305, 6000),
							synthFile("root/src/deep/note.txt", "note.txt", "txt", "Text", 12, 240),
						},
					},
				},
			},
			{
				Path: "root/docs",
				Name: "docs",
				Files: []*model.File{
					synthFile("root/docs/guide.md", "guide.md", "md", "Markdown", 88, 1760),
				},
			},
		},
	}
}
