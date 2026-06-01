package scan

import (
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
)

type filterPolicy struct {
	rootPath string
	rules    []filter.Rule
}

func newFilterPolicy(rootPath string, rules []filter.Rule) filterPolicy {
	return filterPolicy{
		rootPath: rootPath,
		rules:    rules,
	}
}

func (p filterPolicy) includes(path string) (bool, string, error) {
	relPath, err := filepath.Rel(p.rootPath, path)
	if err != nil {
		return false, "", eris.Wrapf(err, "failed to compute relative path for %s", path)
	}

	return filter.IsIncluded(relPath, p.rules), relPath, nil
}
