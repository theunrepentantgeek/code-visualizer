package provider

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

// orderTracker records which providers ran and in what order.
type orderTracker struct {
	mu    sync.Mutex
	calls []metric.Name
}

func (o *orderTracker) record(name metric.Name) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.calls = append(o.calls, name)
}

// mockProvider records load calls and optionally returns an error.
type mockProvider struct {
	name    metric.Name
	kind    metric.Kind
	deps    []metric.Name
	loadErr error
	tracker *orderTracker
}

func (m *mockProvider) Name() metric.Name                 { return m.name }
func (m *mockProvider) Kind() metric.Kind                 { return m.kind }
func (m *mockProvider) Dependencies() []metric.Name       { return m.deps }
func (*mockProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }

func (m *mockProvider) Load(_ *model.Directory) error {
	if m.tracker != nil {
		m.tracker.record(m.name)
	}

	return m.loadErr
}

func TestRunBasicExecution(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	tracker := &orderTracker{}
	reg.register(&mockProvider{name: "m1", kind: metric.Quantity, tracker: tracker})

	err := runWithRegistry(reg, nil, []metric.Name{"m1"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tracker.calls).To(Equal([]metric.Name{"m1"}))
}

func TestRunTransitiveDependencies(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	tracker := &orderTracker{}
	reg.register(&mockProvider{name: "base", kind: metric.Quantity, tracker: tracker})
	reg.register(&mockProvider{name: "derived", kind: metric.Quantity, deps: []metric.Name{"base"}, tracker: tracker})

	err := runWithRegistry(reg, nil, []metric.Name{"derived"})
	g.Expect(err).NotTo(HaveOccurred())

	// "base" must run before "derived"
	g.Expect(tracker.calls).To(HaveLen(2))

	baseIdx := -1
	derivedIdx := -1

	for i, n := range tracker.calls {
		if n == "base" {
			baseIdx = i
		}

		if n == "derived" {
			derivedIdx = i
		}
	}

	g.Expect(baseIdx).To(BeNumerically("<", derivedIdx))
}

func TestRunCycleDetection(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&mockProvider{name: "a", kind: metric.Quantity, deps: []metric.Name{"b"}})
	reg.register(&mockProvider{name: "b", kind: metric.Quantity, deps: []metric.Name{"a"}})

	err := runWithRegistry(reg, nil, []metric.Name{"a"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("circular dependency")))
}

func TestRunUnknownDependency(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&mockProvider{name: "a", kind: metric.Quantity, deps: []metric.Name{"missing"}})

	err := runWithRegistry(reg, nil, []metric.Name{"a"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("unknown metric")))
}

func TestRunUnknownRequestedMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()

	err := runWithRegistry(reg, nil, []metric.Name{"nonexistent"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("unknown metric")))
}

func TestRunUnknownMetricListsAvailable(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&mockProvider{name: "alpha", kind: metric.Quantity})
	reg.register(&mockProvider{name: "beta", kind: metric.Quantity})

	err := runWithRegistry(reg, nil, []metric.Name{"nonexistent"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("available metrics: alpha, beta")))
}

func TestRunErrorPropagation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&mockProvider{name: "fail", kind: metric.Quantity, loadErr: errors.New("load failed")})

	err := runWithRegistry(reg, nil, []metric.Name{"fail"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("load failed")))
}

// concurrentProvider tracks concurrent Load calls via shared atomics.
type concurrentProvider struct {
	name          metric.Name
	counter       *atomic.Int32
	maxConcurrent *atomic.Int32
}

func (c *concurrentProvider) Name() metric.Name                 { return c.name }
func (*concurrentProvider) Kind() metric.Kind                   { return metric.Quantity }
func (*concurrentProvider) Dependencies() []metric.Name         { return nil }
func (*concurrentProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }

func (c *concurrentProvider) Load(_ *model.Directory) error {
	cur := c.counter.Add(1)

	for {
		old := c.maxConcurrent.Load()
		if cur <= old || c.maxConcurrent.CompareAndSwap(old, cur) {
			break
		}
	}

	// Small sleep to keep goroutines alive long enough to overlap
	time.Sleep(10 * time.Millisecond)
	c.counter.Add(-1)

	return nil
}

func TestRunParallelExecution(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()

	var (
		counter       atomic.Int32
		maxConcurrent atomic.Int32
	)

	// Register 3 independent providers — they should run concurrently

	reg.register(&concurrentProvider{name: "p1", counter: &counter, maxConcurrent: &maxConcurrent})
	reg.register(&concurrentProvider{name: "p2", counter: &counter, maxConcurrent: &maxConcurrent})
	reg.register(&concurrentProvider{name: "p3", counter: &counter, maxConcurrent: &maxConcurrent})

	err := runWithRegistry(reg, nil, []metric.Name{"p1", "p2", "p3"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(maxConcurrent.Load()).To(BeNumerically(">", 1), "expected concurrent execution")
}

func TestRunEmptyRequest(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()

	err := runWithRegistry(reg, nil, []metric.Name{})
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRunAutoExpandsDependencies(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	tracker := &orderTracker{}
	reg.register(&mockProvider{name: "base", kind: metric.Quantity, tracker: tracker})
	reg.register(&mockProvider{name: "mid", kind: metric.Quantity, deps: []metric.Name{"base"}, tracker: tracker})
	reg.register(&mockProvider{name: "top", kind: metric.Quantity, deps: []metric.Name{"mid"}, tracker: tracker})

	// Only request "top" — "mid" and "base" should be auto-included
	err := runWithRegistry(reg, nil, []metric.Name{"top"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tracker.calls).To(HaveLen(3))
}
