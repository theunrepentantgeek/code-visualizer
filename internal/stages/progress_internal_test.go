package stages

import (
	"bytes"
	"log/slog"
	"strings"
	"sync/atomic"
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

	g.Expect(buf.String()).To(ContainSubstring(`msg="Loading metrics." loaded=3 total=4 percentage=75`))
	g.Expect(buf.String()).NotTo(ContainSubstring("metric="))
	g.Expect(buf.String()).To(HavePrefix("time="))
	g.Expect(strings.Count(buf.String(), "\n")).To(Equal(1))
}

//nolint:paralleltest // mutates global slog default logger
func TestLogHistoryProgress_LogsAggregateProcessedCommits(t *testing.T) {
	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	oldDefault := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	counter := &atomic.Int64{}
	counter.Store(3)

	logHistoryProgress(counter)

	g.Expect(buf.String()).To(ContainSubstring(`msg="Loading history..." commits=3`))
	g.Expect(buf.String()).To(HavePrefix("time="))
	g.Expect(strings.Count(buf.String(), "\n")).To(Equal(1))
}
