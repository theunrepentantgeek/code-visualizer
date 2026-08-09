package git

import (
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

type metricsLoader struct {
	onFile func()
	mu     sync.Mutex
}

func (l *metricsLoader) SetOnFileProcessed(fn func()) { l.onFile = fn }
func (l *metricsLoader) FileProgressMutex() *sync.Mutex {
	return &l.mu
}

func (l *metricsLoader) Load(root *model.Directory, requested []metric.Name) error {
	return loadGitMetrics(root, requested, l.onFile)
}

// loadAllFileMetrics runs the git analysis once and populates all 7 file-level
// git metrics in a single pass. This replaces 7 separate legacy providers that
// each independently walked git history.
func loadAllFileMetrics(root *model.Directory) error {
	return loadGitMetrics(root, fileMetricNames, nil)
}

type metricRequirements struct {
	processors     []providerDef
	needsLineStats bool
}

func newMetricRequirements(requested []metric.Name) metricRequirements {
	requirements := metricRequirements{
		processors: make([]providerDef, 0, len(requested)),
	}

	for _, name := range requested {
		def, ok := providerDefs[name]
		if !ok {
			continue
		}

		requirements.processors = append(requirements.processors, def)
		if name == TotalLinesAdded || name == TotalLinesRemoved {
			requirements.needsLineStats = true
		}
	}

	return requirements
}

// loadGitMetrics opens the repo service, prewarms data for all scanned files,
// then invokes only requested metric processors for each file.
//
// Git metrics have no silent fallback: if the repository cannot be opened, has
// no history, or contains none of the scanned files, loadGitMetrics returns
// an error rather than producing an empty result that would cascade into
// confusing downstream failures.
func loadGitMetrics(root *model.Directory, requested []metric.Name, onFile func()) error {
	s, err := getService(root.Path)
	if err != nil {
		return eris.Wrapf(err, "git loader requires a git repository")
	}

	requirements := newMetricRequirements(requested)

	pathSet := buildRelPathSet(s, root)
	if err := s.bulkPrewarm(pathSet, requirements, onFile); err != nil {
		return eris.Wrapf(err, "git loader requires readable git history at %s", s.RepoRoot())
	}

	model.WalkFiles(root, func(f *model.File) {
		relPath, relErr := filepath.Rel(s.RepoRoot(), f.Path)
		if relErr != nil {
			slog.Warn("could not compute relative path", "path", f.Path, "error", relErr)

			return
		}

		for _, def := range requirements.processors {
			def.process(s, f, relPath)
		}
	})

	if !s.anyPathHasGitHistory(pathSet) {
		return eris.Errorf(
			"git loader produced no metrics: none of the scanned files under %s have git history",
			s.RepoRoot(),
		)
	}

	return nil
}
