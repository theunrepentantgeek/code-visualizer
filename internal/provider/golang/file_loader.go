package golang

import (
	"io/fs"
	"log/slog"
	"path"
	"path/filepath"
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
		return nil, eris.Wrap(err, "reading Go file metrics")
	}

	moduleFS := f.Source

	moduleDir := path.Dir(f.SourcePath)
	if f.RepoSource != nil {
		moduleFS = f.RepoSource
		moduleDir = path.Dir(f.RepoPath)
	}

	modulePath := findModulePathFS(moduleFS, moduleDir)
	if modulePath == "" && f.RepoSource == nil {
		modulePath = globalModuleCache.findModulePath(filepath.Dir(f.Path))
	}

	return analyzeSource(f.Path, src, modulePath)
}

func findModulePathFS(fsys fs.FS, start string) string {
	for dir := path.Clean(start); ; dir = path.Dir(dir) {
		if module := modulePathAt(fsys, dir); module != "" {
			return module
		}

		if dir == "." {
			return ""
		}
	}
}

func modulePathAt(fsys fs.FS, dir string) string {
	name := "go.mod"
	if dir != "." {
		name = path.Join(dir, name)
	}

	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return ""
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
