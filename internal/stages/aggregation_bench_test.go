package stages_test

import (
	"fmt"
	"testing"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// BenchmarkComputeAggregations measures how long ComputeAggregations takes on a
// synthetic tree roughly representative of a mid-sized Go repository:
// 10 subdirectories, each with 20 files, each file with 5 declarations and 10 commits.
func BenchmarkComputeAggregations(b *testing.B) {
	const (
		numDirs        = 10
		filesPerDir    = 20
		declsPerFile   = 5
		commitsPerFile = 10
	)

	root := buildBenchTree(numDirs, filesPerDir, declsPerFile, commitsPerFile)

	metrics := []provider.ResolvedMetric{
		resolveMetricForBench(b, "file-size.sum"),
		resolveMetricForBench(b, "file-size.mean"),
		resolveMetricForBench(b, "file-size.max"),
		resolveMetricForBench(b, "file-type.mode"),
	}

	b.ResetTimer()

	for range b.N {
		if err := stages.ComputeAggregations(root, metrics); err != nil {
			b.Fatalf("ComputeAggregations: %v", err)
		}
	}
}

func resolveMetricForBench(b *testing.B, raw string) provider.ResolvedMetric {
	b.Helper()

	expr, err := metric.ParseExpression(raw)
	if err != nil {
		b.Fatalf("ParseExpression(%q): %v", raw, err)
	}

	resolved, err := provider.ResolveExpression(expr, metric.LevelDirectory)
	if err != nil {
		b.Fatalf("ResolveExpression(%q): %v", raw, err)
	}

	return resolved
}

func buildBenchTree(numDirs, filesPerDir, declsPerFile, commitsPerFile int) *model.Directory {
	root := &model.Directory{Name: "root"}

	for d := range numDirs {
		sub := &model.Directory{Name: fmt.Sprintf("pkg%d", d)}

		for f := range filesPerDir {
			file := &model.File{Name: fmt.Sprintf("file%d.go", f)}
			file.SetQuantity("file-size", int64((d*filesPerDir+f+1)*100))
			file.SetClassification("file-type", "go")

			for k := range declsPerFile {
				file.Declarations = append(file.Declarations, &model.Declaration{
					Name:       fmt.Sprintf("Func%d", k),
					Kind:       "function",
					Visibility: "public",
				})
			}

			for c := range commitsPerFile {
				file.Commits = append(file.Commits, &model.Commit{
					Hash: fmt.Sprintf("abc%04d", c),
				})
			}

			sub.Files = append(sub.Files, file)
		}

		root.Dirs = append(root.Dirs, sub)
	}

	return root
}
