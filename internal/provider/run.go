package provider

import (
	"github.com/rotisserie/eris"
	"golang.org/x/sync/errgroup"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
)

// Run loads the requested metrics (plus transitive dependencies) onto the tree.
// Providers run in parallel where dependency ordering allows.
func Run(root *model.Directory, requested []metric.Name) error {
	return runWithRegistry(globalRegistry, root, requested)
}

func runWithRegistry(reg *registry, root *model.Directory, requested []metric.Name) error {
	if len(requested) == 0 {
		return nil
	}

	expanded, err := expandDeps(reg, requested)
	if err != nil {
		return err
	}

	levels, err := topoSort(reg, expanded)
	if err != nil {
		return err
	}

	for _, level := range levels {
		g := new(errgroup.Group)

		for _, name := range level {
			p, _ := reg.get(name)

			g.Go(func() error {
				return p.Load(root)
			})
		}

		if err := g.Wait(); err != nil {
			return eris.Wrap(err, "provider load failed")
		}
	}

	return nil
}

// expandDeps returns the transitive closure of requested metric names.
func expandDeps(reg *registry, requested []metric.Name) ([]metric.Name, error) {
	seen := make(map[metric.Name]bool)
	var result []metric.Name

	var visit func(metric.Name) error
	visit = func(name metric.Name) error {
		if seen[name] {
			return nil
		}

		p, ok := reg.get(name)
		if !ok {
			return eris.Errorf("unknown metric %q — no provider registered", name)
		}

		seen[name] = true
		result = append(result, name)

		for _, dep := range p.Dependencies() {
			if err := visit(dep); err != nil {
				return err
			}
		}

		return nil
	}

	for _, name := range requested {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// topoSort groups metrics into execution levels. Each level's metrics have
// all dependencies satisfied by previous levels.
func topoSort(reg *registry, names []metric.Name) ([][]metric.Name, error) {
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
		p, _ := reg.get(n)

		for _, dep := range p.Dependencies() {
			if nameSet[dep] {
				inDegree[n]++
				dependents[dep] = append(dependents[dep], n)
			}
		}
	}

	var levels [][]metric.Name
	processed := 0

	for processed < len(names) {
		var level []metric.Name

		for _, n := range names {
			if inDegree[n] == 0 {
				level = append(level, n)
			}
		}

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