package stages_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

const progressMetric metric.Name = "test-progress"

type progressLoader struct {
	onFile          func()
	mu              sync.Mutex
	err             error
	pauseBeforeLast time.Duration
	ran             *atomic.Bool
}

func (l *progressLoader) SetOnFileProcessed(fn func()) {
	l.onFile = fn
}

func (l *progressLoader) FileProgressMutex() *sync.Mutex {
	return &l.mu
}

func (l *progressLoader) Load(_ context.Context, root *model.Directory, _ []metric.Name) error {
	if l.ran != nil {
		l.ran.Store(true)
	}

	for i := range root.Files {
		if i == len(root.Files)-1 {
			time.Sleep(l.pauseBeforeLast)
		}

		l.onFile()
	}

	return l.err
}

func registerProgressLoader(t *testing.T, loadErr error) {
	t.Helper()

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(func() {
		provider.ResetBaseRegistryForTesting()
		filesystem.Register()
		git.Register()
	})

	loader := &progressLoader{err: loadErr}
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:  []metric.Name{progressMetric},
		Load:     loader.Load,
		Reporter: loader,
	})
}

func progressState() *stages.CommonState {
	return &stages.CommonState{
		Flags: &stages.Flags{},
		Root: &model.Directory{
			Files: []*model.File{{}, {}},
		},
		Requested: stages.RequestedMetrics{
			BaseMetrics: []metric.Name{progressMetric},
		},
	}
}

//nolint:paralleltest // mutates the global provider registry
func TestRunProvidersReportsCompletedMetricProgress(t *testing.T) {
	g := NewGomegaWithT(t)
	registerProgressLoader(t, nil)
	sink := newTestSink(progress.WorkObservations)

	g.Expect(stages.RunProviders(progressState(), context.Background(), sink)).To(Succeed())
	g.Expect(sink.totals).To(Equal([]int64{2}))
	g.Expect(sink.current).To(Equal([]int64{1, 2}))
}

//nolint:paralleltest // mutates the global provider registry
func TestRunProvidersReportsOnlyWorkRemainingAfterGitPrewarm(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(func() {
		provider.ResetBaseRegistryForTesting()
		filesystem.Register()
		git.Register()
	})

	remaining := &progressLoader{}
	prewarmedGitRan := &atomic.Bool{}
	prewarmedGit := &progressLoader{ran: prewarmedGitRan}

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:      []metric.Name{progressMetric},
		Dependencies: []metric.Name{git.FileFreshness},
		Load:         remaining.Load,
		Reporter:     remaining,
	})
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:  []metric.Name{git.FileFreshness},
		Load:     prewarmedGit.Load,
		Reporter: prewarmedGit,
	})

	files := make([]*model.File, 10)
	for i := range files {
		files[i] = &model.File{}
	}

	state := &stages.CommonState{
		Flags:      &stages.Flags{},
		Root:       &model.Directory{Files: files},
		GitHistory: []git.Commit{{Hash: "prewarmed"}},
		Requested: stages.RequestedMetrics{
			BaseMetrics: []metric.Name{progressMetric, git.FileFreshness},
		},
	}

	sink := newTestSink(progress.WorkObservations)
	g.Expect(stages.RunProviders(state, context.Background(), sink)).To(Succeed())

	g.Expect(prewarmedGitRan.Load()).To(BeTrue())
	g.Expect(sink.totals).To(Equal([]int64{10}))
	g.Expect(sink.current).To(HaveLen(10))
}

//nolint:paralleltest // mutates the global provider registry
func TestRunProvidersOmitsCompletionWhenLoadingFailsAtTotal(t *testing.T) {
	g := NewGomegaWithT(t)
	registerProgressLoader(t, errors.New("load failed after reporting progress"))
	sink := newTestSink(progress.WorkObservations)

	err := stages.RunProviders(progressState(), context.Background(), sink)

	g.Expect(err).To(MatchError(ContainSubstring("load failed after reporting progress")))
	g.Expect(sink.current).To(Equal([]int64{1, 2}))
}

//nolint:paralleltest // mutates the global provider registry
func TestRunProvidersPropagatesCancellation(t *testing.T) {
	g := NewGomegaWithT(t)
	registerProgressLoader(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := stages.RunProviders(progressState(), ctx, newTestSink(progress.WorkObservations))

	g.Expect(err).To(MatchError(context.Canceled))
}
