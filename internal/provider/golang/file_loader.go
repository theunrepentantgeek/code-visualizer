package golang

import (
	"io/fs"
	"log/slog"
	"path"
	"runtime"
	"strings"

	"github.com/rotisserie/eris"
	"golang.org/x/sync/errgroup"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

var (
	stdlibImportsMetric   = metric.MetricExpression{Filter: filterStdlib, Base: Imports}.ResultName()
	externalImportsMetric = metric.MetricExpression{Filter: filterExternal, Base: Imports}.ResultName()
	internalImportsMetric = metric.MetricExpression{Filter: filterInternal, Base: Imports}.ResultName()
)

// loadFileMetrics populates file-level Go metrics (imports, comment-ratio)
// and filtered import variants (stdlib.imports, external.imports, internal.imports).
func loadFileMetrics(root *model.Directory) error {
	var files []*model.File

	model.WalkFiles(root, func(f *model.File) {
		files = append(files, f)
	})

	g := new(errgroup.Group)
	g.SetLimit(runtime.NumCPU())

	for _, f := range files {
		g.Go(func() error {
			populateFileMetrics(f)

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return eris.Wrap(err, "loading Go file metrics")
	}

	return nil
}

func populateFileMetrics(f *model.File) {
	if f.Extension != "go" {
		return
	}

	stats, err := analyzeModelFile(f)
	if err != nil {
		slog.Warn("could not analyze Go file for metrics", "path", f.Path, "error", err)

		return
	}

	f.SetQuantity(Imports, stats.imports)
	f.SetQuantity(stdlibImportsMetric, stats.stdlibImports)
	f.SetQuantity(externalImportsMetric, stats.externalImports)
	f.SetQuantity(internalImportsMetric, stats.internalImports)
	f.SetMeasure(CommentRatio, stats.commentRatio)
}

func analyzeModelFile(f *model.File) (*fileStats, error) {
	if f.Source == nil {
		return getOrAnalyze(f.Path)
	}

	src, err := f.ReadAll()
	if err != nil {
		return nil, err
	}

	return analyzeSource(f.Path, src, findModulePathFS(f.Source, path.Dir(f.SourcePath)))
}

func findModulePathFS(fsys fs.FS, start string) string {
	for dir := path.Clean(start); ; dir = path.Dir(dir) {
		name := "go.mod"
		if dir != "." {
			name = path.Join(dir, name)
		}

		data, err := fs.ReadFile(fsys, name)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if value, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
					return strings.TrimSpace(value)
				}
			}
		}
		if dir == "." {
			return ""
		}
	}
}
