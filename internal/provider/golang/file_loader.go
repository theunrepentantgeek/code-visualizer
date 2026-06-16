package golang

import (
	"log/slog"
	"runtime"

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

	stats, err := getOrAnalyze(f.Path)
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
