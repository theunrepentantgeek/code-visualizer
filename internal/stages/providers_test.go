package stages_test

import (
	"bytes"
	"errors"
	"log/slog"
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

type failingFileProgressLoader struct {
	onFile func()
	mu     sync.Mutex
}

func (l *failingFileProgressLoader) SetOnFileProcessed(fn func()) { l.onFile = fn }
func (l *failingFileProgressLoader) FileProgressMutex() *sync.Mutex {
	return &l.mu
}

func (l *failingFileProgressLoader) Load(_ *model.Directory, _ []metric.Name) error {
	l.onFile()
	l.onFile()

	return errors.New("load failed after reporting progress")
}

//nolint:paralleltest // mutates the global provider registry and slog logger
func TestRunProvidersOmitsCompletionWhenLoaderFailsAtTotal(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(func() {
		provider.ResetBaseRegistryForTesting()
		filesystem.Register()
		git.Register()
	})

	loader := &failingFileProgressLoader{}
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:  []metric.Name{"failing-progress"},
		Load:     loader.Load,
		Reporter: loader,
	})

	var buf bytes.Buffer

	oldDefault := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	state := &stages.CommonState{
		Flags: &stages.Flags{},
		Root: &model.Directory{
			Files: []*model.File{{}, {}},
		},
		Requested: stages.RequestedMetrics{
			BaseMetrics: []metric.Name{"failing-progress"},
		},
	}

	err := stages.RunProviders(state)

	g.Expect(err).To(MatchError(ContainSubstring("load failed after reporting progress")))
	g.Expect(buf.String()).To(ContainSubstring(`msg="Loading metrics." loaded=0/2 percentage=0.0`))
	g.Expect(buf.String()).NotTo(ContainSubstring(`msg="Loaded metrics"`))
}
