package git

import (
	"log/slog"
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

type fileProgressCallbacks struct {
	onPrewarm func()
	onFinish  func()
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

	return s.loadGitMetrics(root, requested, onFile)
}

func (s *repoService) loadGitMetrics(
	root *model.Directory,
	requested []metric.Name,
	onFile func(),
) error {
	requirements := newMetricRequirements(requested)

	pathSet := buildRelPathSet(s, root)
	commitTotal := int64(0)

	if onFile != nil {
		missing, _ := s.bulkPrewarmWork(pathSet, requirements)
		if len(missing) > 0 {
			total, err := s.commitTotal()
			if err != nil {
				return eris.Wrap(err, "failed to count git commits")
			}

			commitTotal = total
		}
	}

	progressCallbacks := newFileProgressCallbacks(onFile, int64(model.CountFiles(root)), commitTotal)

	if err := s.bulkPrewarm(pathSet, requirements, progressCallbacks.onPrewarm); err != nil {
		return eris.Wrapf(err, "git loader requires readable git history at %s", s.RepoRoot())
	}

	s.applySelectedFileMetrics(root, requirements)

	if err := s.requireGitHistory(pathSet); err != nil {
		return err
	}

	if progressCallbacks.onFinish != nil {
		progressCallbacks.onFinish()
	}

	return nil
}

func newFileProgressCallbacks(onFile func(), fileTotal int64, commitTotal int64) fileProgressCallbacks {
	if onFile == nil || fileTotal == 0 {
		return fileProgressCallbacks{}
	}

	var (
		commitsProcessed int64
		filesReported    int64
	)

	reportThrough := func(target int64) {
		for filesReported < target {
			onFile()

			filesReported++
		}
	}

	return fileProgressCallbacks{
		onPrewarm: func() {
			commitsProcessed++
			if commitTotal > 0 {
				target := commitsProcessed * (fileTotal - 1) / commitTotal
				reportThrough(min(target, fileTotal-1))
			}
		},
		onFinish: func() {
			reportThrough(fileTotal)
		},
	}
}

func (s *repoService) applySelectedFileMetrics(
	root *model.Directory,
	requirements metricRequirements,
) {
	model.WalkFiles(root, func(f *model.File) {
		relPath, relErr := repoRelativePath(s.RepoRoot(), f.Path)
		if relErr != nil {
			slog.Warn("could not compute relative path", "path", f.Path, "error", relErr)

			return
		}

		for _, def := range requirements.processors {
			def.process(s, f, relPath)
		}
	})
}

func (s *repoService) requireGitHistory(pathSet map[string]bool) error {
	if s.anyPathHasGitHistory(pathSet) {
		return nil
	}

	return eris.Errorf(
		"git loader produced no metrics: none of the scanned files under %s have git history",
		s.RepoRoot(),
	)
}
