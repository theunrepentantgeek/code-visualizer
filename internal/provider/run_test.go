package provider_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

type loaderOrderTracker struct {
	mu    sync.Mutex
	calls []metric.Name
}

func (t *loaderOrderTracker) record(name metric.Name) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.calls = append(t.calls, name)
}

type progressTracker struct {
	mu       sync.Mutex
	started  []metric.Name
	finished []metric.Name
}

func (t *progressTracker) OnMetricStarted(name metric.Name) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.started = append(t.started, name)
}

func (t *progressTracker) OnMetricFinished(name metric.Name) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.finished = append(t.finished, name)
}

func (*progressTracker) OnFileProcessed(metric.Name) {}

func resetBaseRegistry(t *testing.T) {
	t.Helper()

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersBasicExecution(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	tracker := &loaderOrderTracker{}
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{"m1"},
		Load: func(_ *model.Directory) error {
			tracker.record("m1")
			return nil
		},
	})

	err := provider.RunLoaders(nil, []metric.Name{"m1"}, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tracker.calls).To(Equal([]metric.Name{"m1"}))
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersRespectsDependencies(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	tracker := &loaderOrderTracker{}
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{"base"},
		Load: func(_ *model.Directory) error {
			tracker.record("base")
			return nil
		},
	})
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:      []metric.Name{"derived"},
		Dependencies: []metric.Name{"base"},
		Load: func(_ *model.Directory) error {
			tracker.record("derived")
			return nil
		},
	})

	err := provider.RunLoaders(nil, []metric.Name{"base", "derived"}, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tracker.calls).To(Equal([]metric.Name{"base", "derived"}))
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersCycleDetection(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:      []metric.Name{"a"},
		Dependencies: []metric.Name{"b"},
		Load:         func(_ *model.Directory) error { return nil },
	})
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:      []metric.Name{"b"},
		Dependencies: []metric.Name{"a"},
		Load:         func(_ *model.Directory) error { return nil },
	})

	err := provider.RunLoaders(nil, []metric.Name{"a", "b"}, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("circular dependency detected among metric loaders")))
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersErrorPropagation(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{"fail"},
		Load: func(_ *model.Directory) error {
			return errors.New("load failed")
		},
	})

	err := provider.RunLoaders(nil, []metric.Name{"fail"}, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("loader level failed")))
	g.Expect(err).To(MatchError(ContainSubstring("load failed")))
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersParallelExecution(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	var (
		counter       atomic.Int32
		maxConcurrent atomic.Int32
	)

	registerConcurrentLoader := func(name metric.Name) {
		provider.RegisterLoader(provider.BaseMetricLoader{
			Metrics: []metric.Name{name},
			Load: func(_ *model.Directory) error {
				current := counter.Add(1)
				for {
					max := maxConcurrent.Load()
					if current <= max || maxConcurrent.CompareAndSwap(max, current) {
						break
					}
				}

				time.Sleep(10 * time.Millisecond)
				counter.Add(-1)

				return nil
			},
		})
	}

	registerConcurrentLoader("p1")
	registerConcurrentLoader("p2")
	registerConcurrentLoader("p3")

	err := provider.RunLoaders(nil, []metric.Name{"p1", "p2", "p3"}, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(maxConcurrent.Load()).To(BeNumerically(">", 1))
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersReportsProgress(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	progress := &progressTracker{}
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{"m1", "m2"},
		Load:    func(_ *model.Directory) error { return nil },
	})

	err := provider.RunLoaders(nil, []metric.Name{"m1"}, progress)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(progress.started).To(Equal([]metric.Name{"m1", "m2"}))
	g.Expect(progress.finished).To(Equal([]metric.Name{"m1", "m2"}))
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersEmptyRequest(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	err := provider.RunLoaders(nil, nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
}
