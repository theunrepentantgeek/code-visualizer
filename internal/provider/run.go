package provider

import (
	"github.com/rotisserie/eris"
	"golang.org/x/sync/errgroup"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// MetricProgress receives notifications as metrics are calculated.
// Callbacks may be called concurrently when loaders run in parallel.
type MetricProgress interface {
	OnMetricStarted(name metric.Name)
	OnMetricFinished(name metric.Name)
	OnFileProcessed(name metric.Name)
}

// FileProgressReporter is retained for loader adapters that can surface
// per-file progress while a loader runs.
type FileProgressReporter interface {
	SetOnFileProcessed(fn func())
}

// RunLoaders loads the requested base metrics using registered loaders.
// Loaders run in parallel where dependency ordering allows.
func RunLoaders(root *model.Directory, requested []metric.Name, progress MetricProgress) error {
	loaders := LoadersFor(requested)
	if len(loaders) == 0 {
		return nil
	}

	levels, err := topoSortLoaders(loaders)
	if err != nil {
		return err
	}

	for _, level := range levels {
		if err := runLoaderLevel(root, level, progress); err != nil {
			return err
		}
	}

	return nil
}

func runLoaderLevel(root *model.Directory, level []BaseMetricLoader, progress MetricProgress) error {
	g := new(errgroup.Group)

	for _, loader := range level {
		g.Go(func() error {
			return runSingleLoader(root, loader, progress)
		})
	}

	if err := g.Wait(); err != nil {
		return eris.Wrap(err, "loader level failed")
	}

	return nil
}

func runSingleLoader(root *model.Directory, loader BaseMetricLoader, progress MetricProgress) error {
	for _, m := range loader.Metrics {
		if progress != nil {
			progress.OnMetricStarted(m)
		}
	}

	if loader.Reporter != nil && progress != nil {
		loader.Reporter.SetOnFileProcessed(func() {
			for _, m := range loader.Metrics {
				progress.OnFileProcessed(m)
			}
		})
	}

	if err := loader.Load(root); err != nil {
		return eris.Wrapf(err, "loader failed for metrics %v", loader.Metrics)
	}

	for _, m := range loader.Metrics {
		if progress != nil {
			progress.OnMetricFinished(m)
		}
	}

	return nil
}

func topoSortLoaders(loaders []BaseMetricLoader) ([][]BaseMetricLoader, error) {
	provides := make(map[metric.Name]int)

	for i, l := range loaders {
		for _, m := range l.Metrics {
			provides[m] = i
		}
	}

	inDegree := make([]int, len(loaders))
	dependents := make(map[int][]int)

	for i, l := range loaders {
		for _, dep := range l.Dependencies {
			if j, ok := provides[dep]; ok && j != i {
				inDegree[i]++
				dependents[j] = append(dependents[j], i)
			}
		}
	}

	return computeLoaderLevels(loaders, inDegree, dependents)
}

func computeLoaderLevels(
	loaders []BaseMetricLoader,
	inDegree []int,
	dependents map[int][]int,
) ([][]BaseMetricLoader, error) {
	var levels [][]BaseMetricLoader

	processed := 0

	for processed < len(loaders) {
		level, levelIndices := findReadyLoaders(loaders, inDegree)

		if len(level) == 0 {
			return nil, eris.New("circular dependency detected among metric loaders")
		}

		processed += advanceLoaderLevel(levelIndices, inDegree, dependents)

		levels = append(levels, level)
	}

	return levels, nil
}

func findReadyLoaders(loaders []BaseMetricLoader, inDegree []int) ([]BaseMetricLoader, []int) {
	var (
		level   []BaseMetricLoader
		indices []int
	)

	for i, deg := range inDegree {
		if deg == 0 {
			level = append(level, loaders[i])
			indices = append(indices, i)
		}
	}

	return level, indices
}

func advanceLoaderLevel(levelIndices []int, inDegree []int, dependents map[int][]int) int {
	for _, i := range levelIndices {
		inDegree[i] = -1

		for _, dep := range dependents[i] {
			inDegree[dep]--
		}
	}

	return len(levelIndices)
}
