package stages

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

//nolint:paralleltest // mutates global slog default logger
func TestLogMetricProgress_LogsAggregateLoadedObservations(t *testing.T) {
	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	oldDefault := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	tracker := &metricProgressTracker{total: 4}
	tracker.OnMetricStarted("first")
	tracker.OnMetricStarted("second")
	tracker.OnFileProcessed("first")
	tracker.OnFileProcessed("first")
	tracker.OnFileProcessed("second")

	logMetricProgress(tracker)

	g.Expect(buf.String()).To(ContainSubstring(`msg="Loading metrics." loaded=3/4 percentage=75.0`))
	g.Expect(buf.String()).NotTo(ContainSubstring("metric="))
	g.Expect(buf.String()).To(HavePrefix("time="))
	g.Expect(strings.Count(buf.String(), "\n")).To(Equal(1))
}

//nolint:paralleltest // mutates global slog default logger
func TestBuildMetricProgressLogsInitialZeroProgress(t *testing.T) {
	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	oldDefault := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	_, stop := BuildMetricProgress(&Flags{}, 4)
	stop(false)

	g.Expect(buf.String()).To(ContainSubstring(`msg="Loading metrics." loaded=0/4 percentage=0.0`))
	g.Expect(strings.Count(buf.String(), "\n")).To(Equal(1))
}

//nolint:paralleltest // mutates global slog default logger
func TestBuildMetricProgressLogsCompletionAtTotal(t *testing.T) {
	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	oldDefault := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	progress, stop := BuildMetricProgress(&Flags{}, 4)
	g.Expect(progress).NotTo(BeNil())

	if progress != nil {
		for range 4 {
			progress.OnFileProcessed("file-lines")
		}
	}

	stop(true)

	g.Expect(buf.String()).To(ContainSubstring(`msg="Loaded metrics" loaded=4/4 percentage=100.0`))
	g.Expect(strings.Count(buf.String(), `msg="Loaded metrics"`)).To(Equal(1))
	g.Expect(strings.TrimSpace(buf.String())).To(HaveSuffix(`msg="Loaded metrics" loaded=4/4 percentage=100.0`))
}

//nolint:paralleltest // mutates global slog default logger
func TestBuildMetricProgressOmitsCompletionAfterFailureAtTotal(t *testing.T) {
	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	oldDefault := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	progress, stop := BuildMetricProgress(&Flags{}, 4)
	g.Expect(progress).NotTo(BeNil())

	if progress != nil {
		for range 4 {
			progress.OnFileProcessed("file-lines")
		}
	}

	stop(false)

	g.Expect(buf.String()).NotTo(ContainSubstring(`msg="Loaded metrics"`))
}

//nolint:paralleltest // mutates global slog default logger
func TestBuildMetricProgressLogsSuccessfulZeroWorkCompletion(t *testing.T) {
	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	oldDefault := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	_, stop := BuildMetricProgress(&Flags{}, 0)
	stop(true)

	g.Expect(buf.String()).To(ContainSubstring(`msg="Loaded metrics" loaded=0/0 percentage=100.0`))
}

//nolint:paralleltest // mutates global slog default logger
func TestLogHistoryProgress_LogsAggregateProcessedCommits(t *testing.T) {
	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	oldDefault := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	tracker := &historyProgressTracker{total: 4}
	tracker.loaded.Store(3)

	logHistoryProgress(tracker)

	g.Expect(buf.String()).To(ContainSubstring(`msg="Loading history." commits=3/4 percentage=75.0`))
	g.Expect(buf.String()).To(HavePrefix("time="))
	g.Expect(strings.Count(buf.String(), "\n")).To(Equal(1))
}
