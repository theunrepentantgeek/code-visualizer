package provider

import (
	"strings"

	"github.com/rotisserie/eris"
	"golang.org/x/sync/errgroup"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// MetricProgress receives notifications as metrics are calculated.
// Callbacks may be called concurrently when providers run in parallel.
type MetricProgress interface {
	OnMetricStarted(name metric.Name)
	OnMetricFinished(name metric.Name)
	OnFileProcessed(name metric.Name)
}

// FileProgressReporter is optionally implemented by providers that report
// per-file progress during Load. runProvider sets the callback before Load.
type FileProgressReporter interface {
	SetOnFileProcessed(fn func())
}

// Run loads the requested metrics (plus transitive dependencies) onto the tree.
// Providers run in parallel where dependency ordering allows.
func Run(root *model.Directory, requested []metric.Name, target metric.Target, progress MetricProgress) error {
	return runWithRegistry(globalRegistry, root, requested, target, progress)
}

func runWithRegistry(
	reg *registry,
	root *model.Directory,
	requested []metric.Name,
	target metric.Target,
	progress MetricProgress,
) error {
	if len(requested) == 0 {
		return nil
	}

	expanded, err := expandDeps(reg, requested, target)
	if err != nil {
		return err
	}

	levels, err := topoSort(reg, expanded, target)
	if err != nil {
		return err
	}

	for _, level := range levels {
		g := new(errgroup.Group)

		for _, name := range level {
			p, _ := reg.get(name, target)

			g.Go(func() error {
				return runProvider(p, root, name, progress)
			})
		}

		if err := g.Wait(); err != nil {
			return err //nolint:wrapcheck // error is wrapped inside runProvider
		}
	}

	return nil
}

// runProvider executes a single provider, notifying progress before and after.
func runProvider(p Interface, root *model.Directory, name metric.Name, progress MetricProgress) error {
	if progress != nil {
		progress.OnMetricStarted(name)
	}

	if reporter, ok := p.(FileProgressReporter); ok && progress != nil {
		reporter.SetOnFileProcessed(func() {
			progress.OnFileProcessed(name)
		})
	}

	if err := p.Load(root); err != nil {
		return eris.Wrapf(err, "provider load failed for metric %q", name)
	}

	if progress != nil {
		progress.OnMetricFinished(name)
	}

	return nil
}

// expandDeps returns the transitive closure of requested metric names.
func expandDeps(reg *registry, requested []metric.Name, target metric.Target) ([]metric.Name, error) {
	seen := make(map[metric.Name]bool)

	var result []metric.Name

	for _, name := range requested {
		if err := visitDep(reg, name, seen, &result, target); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func visitDep(
	reg *registry,
	name metric.Name,
	seen map[metric.Name]bool,
	result *[]metric.Name,
	target metric.Target,
) error {
	if seen[name] {
		return nil
	}

	p, ok := reg.get(name, target)
	if !ok || p == nil {
		return metricNotFoundError(reg, name, target)
	}

	seen[name] = true
	*result = append(*result, name)

	for _, dep := range p.Dependencies() {
		if err := visitDep(reg, dep, seen, result, target); err != nil {
			return err
		}
	}

	return nil
}

// metricNotFoundError builds an error for a missing metric, including a hint
// if the metric exists for a different target.
func metricNotFoundError(reg *registry, name metric.Name, target metric.Target) error {
	targets := reg.targetsForName(name)
	if len(targets) > 0 {
		targetStrs := make([]string, len(targets))
		for i, t := range targets {
			targetStrs[i] = t.String()
		}

		return eris.Errorf(
			"unknown %s metric %q; metric %q exists for target(s): %s",
			target, name, name, strings.Join(targetStrs, ", "),
		)
	}

	return eris.Errorf(
		"unknown %s metric %q; available metrics: %s",
		target,
		name,
		formatNames(reg.namesFor(target)),
	)
}

// topoSort groups metrics into execution levels. Each level's metrics have
// all dependencies satisfied by previous levels.
func topoSort(reg *registry, names []metric.Name, target metric.Target) ([][]metric.Name, error) {
	inDegree, dependents := buildDepGraph(reg, names, target)

	return computeLevels(names, inDegree, dependents)
}

func buildDepGraph(
	reg *registry,
	names []metric.Name,
	target metric.Target,
) (map[metric.Name]int, map[metric.Name][]metric.Name) {
	nameSet := make(map[metric.Name]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}

	inDegree := make(map[metric.Name]int, len(names))
	dependents := make(map[metric.Name][]metric.Name)

	for _, n := range names {
		inDegree[n] = 0
	}

	for _, n := range names {
		addEdges(reg, target, n, nameSet, inDegree, dependents)
	}

	return inDegree, dependents
}

func addEdges(
	reg *registry,
	target metric.Target,
	n metric.Name,
	nameSet map[metric.Name]bool,
	inDegree map[metric.Name]int,
	dependents map[metric.Name][]metric.Name,
) {
	p, ok := reg.get(n, target)
	if !ok || p == nil {
		return
	}

	for _, dep := range p.Dependencies() {
		if nameSet[dep] {
			inDegree[n]++
			dependents[dep] = append(dependents[dep], n)
		}
	}
}

func computeLevels(
	names []metric.Name,
	inDegree map[metric.Name]int,
	dependents map[metric.Name][]metric.Name,
) ([][]metric.Name, error) {
	var levels [][]metric.Name

	processed := 0

	for processed < len(names) {
		level := findReady(names, inDegree)

		if len(level) == 0 {
			return nil, eris.New("circular dependency detected among metric providers")
		}

		for _, n := range level {
			inDegree[n] = -1
			processed++

			for _, dep := range dependents[n] {
				inDegree[dep]--
			}
		}

		levels = append(levels, level)
	}

	return levels, nil
}

func findReady(names []metric.Name, inDegree map[metric.Name]int) []metric.Name {
	var level []metric.Name

	for _, n := range names {
		if inDegree[n] == 0 {
			level = append(level, n)
		}
	}

	return level
}

func formatNames(names []metric.Name) string {
	strs := make([]string, len(names))
	for i, n := range names {
		strs[i] = string(n)
	}

	return strings.Join(strs, ", ")
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
		g := new(errgroup.Group)
		for _, loader := range level {
			g.Go(func() error {
				for _, m := range loader.Metrics {
					if progress != nil {
						progress.OnMetricStarted(m)
					}
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
			})
		}

		if err := g.Wait(); err != nil {
			return err
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

	var levels [][]BaseMetricLoader
	processed := 0

	for processed < len(loaders) {
		var (
			level        []BaseMetricLoader
			levelIndices []int
		)

		for i, deg := range inDegree {
			if deg == 0 {
				level = append(level, loaders[i])
				levelIndices = append(levelIndices, i)
			}
		}

		if len(level) == 0 {
			return nil, eris.New("circular dependency detected among metric loaders")
		}

		for _, i := range levelIndices {
			inDegree[i] = -1
			processed++
			for _, dep := range dependents[i] {
				inDegree[dep]--
			}
		}

		levels = append(levels, level)
	}

	return levels, nil
}
