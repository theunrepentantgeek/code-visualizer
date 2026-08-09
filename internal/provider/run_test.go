package provider_test

import (
	"errors"
	"slices"
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

type recordingLoader struct {
	requested []metric.Name
}

func (l *recordingLoader) Load(_ *model.Directory, requested []metric.Name) error {
	l.requested = slices.Clone(requested)

	return nil
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
		Load: func(_ *model.Directory, _ []metric.Name) error {
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
		Load: func(_ *model.Directory, _ []metric.Name) error {
			tracker.record("base")

			return nil
		},
	})
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:      []metric.Name{"derived"},
		Dependencies: []metric.Name{"base"},
		Load: func(_ *model.Directory, _ []metric.Name) error {
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
		Load:         func(_ *model.Directory, _ []metric.Name) error { return nil },
	})
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:      []metric.Name{"b"},
		Dependencies: []metric.Name{"a"},
		Load:         func(_ *model.Directory, _ []metric.Name) error { return nil },
	})

	err := provider.RunLoaders(nil, []metric.Name{"a", "b"}, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("circular dependency detected among metric loaders")))
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersFailsOnMissingDependency(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:      []metric.Name{"derived"},
		Dependencies: []metric.Name{"base"},
		Load:         func(_ *model.Directory, _ []metric.Name) error { return nil },
	})

	err := provider.RunLoaders(nil, []metric.Name{"derived"}, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("no selected loader provides it")))
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersErrorPropagation(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{"fail"},
		Load: func(_ *model.Directory, _ []metric.Name) error {
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
			Load: func(_ *model.Directory, _ []metric.Name) error {
				current := counter.Add(1)

				for {
					peak := maxConcurrent.Load()
					if current <= peak || maxConcurrent.CompareAndSwap(peak, current) {
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
func TestRunLoadersPassesRequestedMetricsInLoaderOrder(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	loader := &recordingLoader{}
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{"first", "second", "third"},
		Load:    loader.Load,
	})

	err := provider.RunLoaders(nil, []metric.Name{"third", "first"}, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(loader.requested).To(Equal([]metric.Name{"first", "third"}))
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersReportsProgressForRequestedMetrics(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	progress := &progressTracker{}

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{"first", "second"},
		Load:    func(_ *model.Directory, _ []metric.Name) error { return nil },
	})

	err := provider.RunLoaders(nil, []metric.Name{"second"}, progress)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(progress.started).To(Equal([]metric.Name{"second"}))
	g.Expect(progress.finished).To(Equal([]metric.Name{"second"}))
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersEmptyRequest(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	err := provider.RunLoaders(nil, nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

type fileProgressLoader struct {
	onFile func()
	mu     sync.Mutex
}

func (l *fileProgressLoader) SetOnFileProcessed(fn func()) { l.onFile = fn }
func (l *fileProgressLoader) FileProgressMutex() *sync.Mutex {
	return &l.mu
}

func (l *fileProgressLoader) Load(_ *model.Directory, _ []metric.Name) error {
	if l.onFile != nil {
		l.onFile()
		l.onFile()
	}

	return nil
}

type fileProgressTracker struct {
	progressTracker
	fileProcessed []metric.Name
}

func (t *fileProgressTracker) OnFileProcessed(name metric.Name) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.fileProcessed = append(t.fileProcessed, name)
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersWiresFileProgressReporter(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	progress := &fileProgressTracker{}
	loader := &fileProgressLoader{}

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:  []metric.Name{"lines"},
		Load:     loader.Load,
		Reporter: loader,
	})

	err := provider.RunLoaders(nil, []metric.Name{"lines"}, progress)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(progress.fileProcessed).To(Equal([]metric.Name{"lines", "lines"}))
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersReportsSelectedMetricsForFileProgress(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	progress := &fileProgressTracker{}
	loader := &fileProgressLoader{}

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:  []metric.Name{"a", "b", "c"},
		Load:     loader.Load,
		Reporter: loader,
	})

	err := provider.RunLoaders(nil, []metric.Name{"c"}, progress)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(progress.fileProcessed).To(Equal([]metric.Name{"c", "c"}))
}

type blockingFileProgressLoader struct {
	onFile        func()
	mu            sync.Mutex
	firstSet      chan struct{}
	secondSet     chan struct{}
	continueFirst chan struct{}
	setCount      atomic.Int32
}

func (l *blockingFileProgressLoader) SetOnFileProcessed(fn func()) {
	l.onFile = fn

	if l.setCount.Add(1) == 1 {
		close(l.firstSet)
		<-l.continueFirst

		return
	}

	close(l.secondSet)
}

func (l *blockingFileProgressLoader) FileProgressMutex() *sync.Mutex {
	return &l.mu
}

func (l *blockingFileProgressLoader) Load(_ *model.Directory, _ []metric.Name) error {
	if l.onFile != nil {
		l.onFile()
	}

	return nil
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersSerializesSharedFileProgressReporters(t *testing.T) {
	g := NewGomegaWithT(t)
	resetBaseRegistry(t)

	loader := &blockingFileProgressLoader{
		firstSet:      make(chan struct{}),
		secondSet:     make(chan struct{}),
		continueFirst: make(chan struct{}),
	}
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:  []metric.Name{"first"},
		Load:     loader.Load,
		Reporter: loader,
	})
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:  []metric.Name{"second"},
		Load:     loader.Load,
		Reporter: loader,
	})

	first := &fileProgressTracker{}
	second := &fileProgressTracker{}
	firstDone := make(chan error, 1)
	secondDone := make(chan error, 1)

	go func() {
		firstDone <- provider.RunLoaders(nil, []metric.Name{"first"}, first)
	}()

	<-loader.firstSet

	go func() {
		secondDone <- provider.RunLoaders(nil, []metric.Name{"second"}, second)
	}()

	select {
	case <-loader.secondSet:
		t.Fatal("second progress callback replaced the first before its load completed")
	case <-time.After(10 * time.Millisecond):
	}

	close(loader.continueFirst)

	g.Expect(<-firstDone).To(Succeed())
	<-loader.secondSet
	g.Expect(<-secondDone).To(Succeed())
	g.Expect(first.fileProcessed).To(Equal([]metric.Name{"first"}))
	g.Expect(second.fileProcessed).To(Equal([]metric.Name{"second"}))
}
