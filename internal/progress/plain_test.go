package progress

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/sebdah/goldie/v2"
)

func plainTestReporter(t *testing.T, verbose bool) (Reporter, *bytes.Buffer, *time.Time) {
	t.Helper()

	var output bytes.Buffer

	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	reporter, err := New(Config{
		Mode:       ModePlain,
		Writer:     &output,
		IsTerminal: func(io.Writer) bool { return false },
		LookupEnv:  env(nil),
		Verbose:    verbose,
		Now:        func() time.Time { return now },
	})
	NewWithT(t).Expect(err).NotTo(HaveOccurred())

	if reporter == nil {
		panic("progress.New returned a nil reporter without an error")
	}

	return reporter, &output, &now
}

func TestPlain_SuccessOutputMatchesGolden(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter, output, now := plainTestReporter(t, false)

	g.Expect(reporter.Begin("Processing project", 2)).To(Succeed())
	preparing, err := reporter.StartStage("Preparing inputs", StageSummary, WorkNone)
	g.Expect(err).NotTo(HaveOccurred())

	if preparing == nil {
		panic("StartStage returned a nil stage without an error")
	}

	*now = now.Add(time.Second)

	g.Expect(preparing.Complete()).To(Succeed())

	acquiring, err := reporter.StartStage("Acquiring data", StageLive, WorkObservations)
	g.Expect(err).NotTo(HaveOccurred())

	if acquiring == nil {
		panic("StartStage returned a nil stage without an error")
	}

	g.Expect(acquiring.SetTotal(100)).To(Succeed())
	g.Expect(acquiring.SetProgress(5)).To(Succeed())

	*now = now.Add(10 * time.Second)

	g.Expect(acquiring.SetProgress(6)).To(Succeed())

	*now = now.Add(time.Second)

	g.Expect(acquiring.Complete()).To(Succeed())
	g.Expect(reporter.Finish()).To(Succeed())
	g.Expect(reporter.Close()).To(Succeed())

	goldie.New(t).Assert(t, "plain-success", output.Bytes())
	g.Expect(output.Bytes()).NotTo(ContainSubstring("\x1b"))
}

func TestPlain_FailureOutputMatchesGoldenAndIsLineSafe(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter, output, now := plainTestReporter(t, false)

	g.Expect(reporter.Begin("Processing project", 1)).To(Succeed())
	stage, err := reporter.StartStage("Acquiring data", StageLive, WorkCommits)
	g.Expect(err).NotTo(HaveOccurred())

	if stage == nil {
		panic("StartStage returned a nil stage without an error")
	}

	*now = now.Add(2 * time.Second)

	g.Expect(stage.Fail(errors.New("boom\ndetails"))).To(Succeed())
	g.Expect(reporter.Close()).To(Succeed())

	goldie.New(t).Assert(t, "plain-failure", output.Bytes())
}

func TestPlain_ThrottlesProgressByPercentageAndTime(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter, output, now := plainTestReporter(t, false)

	g.Expect(reporter.Begin("Processing", 1)).To(Succeed())
	stage, err := reporter.StartStage("Work", StageLive, WorkObservations)
	g.Expect(err).NotTo(HaveOccurred())

	if stage == nil {
		panic("StartStage returned a nil stage without an error")
	}

	g.Expect(stage.SetTotal(100)).To(Succeed())
	g.Expect(stage.SetProgress(4)).To(Succeed())
	g.Expect(stage.SetProgress(5)).To(Succeed())
	g.Expect(stage.SetProgress(5)).To(Succeed())

	*now = now.Add(10 * time.Second)

	g.Expect(stage.SetProgress(6)).To(Succeed())
	g.Expect(stage.SetProgress(100)).To(Succeed())

	text := output.String()
	g.Expect(text).NotTo(ContainSubstring("4/100"))
	g.Expect(strings.Count(text, "5/100")).To(Equal(1))
	g.Expect(strings.Count(text, "6/100")).To(Equal(1))
	g.Expect(strings.Count(text, "100/100")).To(Equal(1))
}

func TestPlain_StatusRequiresVerboseMode(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		verbose bool
		want    bool
	}{
		{name: "default", verbose: false, want: false},
		{name: "verbose", verbose: true, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			reporter, output, _ := plainTestReporter(t, tc.verbose)
			g.Expect(reporter.Begin("Processing", 1)).To(Succeed())
			stage, err := reporter.StartStage("Work", StageLive, WorkObservations)
			g.Expect(err).NotTo(HaveOccurred())

			if stage == nil {
				panic("StartStage returned a nil stage without an error")
			}

			g.Expect(stage.SetStatus("reading src/main.go")).To(Succeed())
			g.Expect(strings.Contains(output.String(), "reading src/main.go")).To(Equal(tc.want))
		})
	}
}

func TestPlain_ZeroTotalStageStaysIndeterminate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter, output, _ := plainTestReporter(t, false)

	g.Expect(reporter.Begin("Processing", 1)).To(Succeed())
	stage, err := reporter.StartStage("Work", StageLive, WorkObservations)
	g.Expect(err).NotTo(HaveOccurred())

	if stage == nil {
		panic("StartStage returned a nil stage without an error")
	}

	g.Expect(stage.Complete()).To(Succeed())
	g.Expect(reporter.Finish()).To(Succeed())

	g.Expect(output.String()).NotTo(ContainSubstring("%"))
	g.Expect(output.String()).NotTo(ContainSubstring("/0"))
}

func TestPlain_CancellationIsDistinctFromFailure(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter, output, _ := plainTestReporter(t, false)

	g.Expect(reporter.Begin("Processing", 1)).To(Succeed())
	stage, err := reporter.StartStage("Work", StageLive, WorkCommits)
	g.Expect(err).NotTo(HaveOccurred())

	if stage == nil {
		panic("StartStage returned a nil stage without an error")
	}

	g.Expect(stage.Cancel(errors.New("interrupted"))).To(Succeed())

	g.Expect(output.String()).To(ContainSubstring("[1/1] Work: cancelled"))
	g.Expect(output.String()).NotTo(ContainSubstring("failed"))
}
