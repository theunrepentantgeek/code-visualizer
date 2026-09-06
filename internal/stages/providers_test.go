package stages_test

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

const progressMetric metric.Name = "test-progress"

type progressLoader struct {
	onFile func()
	mu     sync.Mutex
	err    error
}

func (l *progressLoader) SetOnFileProcessed(fn func()) {
	l.onFile = fn
}

func (l *progressLoader) FileProgressMutex() *sync.Mutex {
	return &l.mu
}

func (l *progressLoader) Load(root *model.Directory, _ []metric.Name) error {
	for range root.Files {
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

//nolint:paralleltest // mutates the global provider registry and slog logger
func TestRunProvidersReportsCompletedMetricProgress(t *testing.T) {
	g := NewGomegaWithT(t)
	registerProgressLoader(t, nil)

	var buf bytes.Buffer

	oldDefault := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	g.Expect(stages.RunProviders(progressState())).To(Succeed())

	output := buf.String()
	g.Expect(output).To(ContainSubstring(`msg="Loading metrics." loaded=0/2 percentage=0.0`))
	g.Expect(output).To(ContainSubstring(`msg="Loaded metrics" loaded=2/2 percentage=100.0`))
	g.Expect(strings.LastIndex(output, `msg="Loaded metrics"`)).
		To(BeNumerically(">", strings.LastIndex(output, `msg="Loading metrics."`)))
}

//nolint:paralleltest // mutates the global provider registry and slog logger
func TestRunProvidersOmitsCompletionWhenLoadingFailsAtTotal(t *testing.T) {
	g := NewGomegaWithT(t)
	registerProgressLoader(t, errors.New("load failed after reporting progress"))

	var buf bytes.Buffer

	oldDefault := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	err := stages.RunProviders(progressState())

	g.Expect(err).To(MatchError(ContainSubstring("load failed after reporting progress")))
	g.Expect(buf.String()).NotTo(ContainSubstring(`msg="Loaded metrics"`))
}
