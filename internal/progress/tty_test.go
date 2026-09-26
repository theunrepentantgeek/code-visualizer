package progress

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

type fakeLiveDisplay struct {
	kind    string
	text    string
	current int64
	stops   int
}

func (d *fakeLiveDisplay) updateText(text string) error {
	d.text = text

	return nil
}

func (d *fakeLiveDisplay) setCurrent(current int64) error {
	d.current = current

	return nil
}

func (d *fakeLiveDisplay) stop() error {
	d.stops++

	return nil
}

type fakeTTYBackend struct {
	started []*fakeLiveDisplay
	lines   []string
}

func (b *fakeTTYBackend) startSpinner(text string, _ bool) (liveDisplay, error) {
	display := &fakeLiveDisplay{kind: "spinner", text: text}
	b.started = append(b.started, display)

	return display, nil
}

func (b *fakeTTYBackend) startProgress(text string, _ int64, _ bool) (liveDisplay, error) {
	display := &fakeLiveDisplay{kind: "progress", text: text}
	b.started = append(b.started, display)

	return display, nil
}

func (b *fakeTTYBackend) println(text string) error {
	b.lines = append(b.lines, text)

	return nil
}

func ttyTestReporter(t *testing.T, unicode, noColor bool) (Reporter, *fakeTTYBackend) {
	t.Helper()

	backend := &fakeTTYBackend{}
	config := Config{
		Mode:            ModeTTY,
		Writer:          io.Discard,
		IsTerminal:      func(io.Writer) bool { return true },
		SupportsUnicode: func() bool { return unicode },
		LookupEnv:       env(nil),
		NoColor:         noColor,
		Now:             time.Now,
	}
	resolved, err := resolveConfig(config)
	NewWithT(t).Expect(err).NotTo(HaveOccurred())

	ttyOutput, err := newTTYRenderer(resolved, backend)
	NewWithT(t).Expect(err).NotTo(HaveOccurred())

	return newReporter(config, ttyOutput), backend
}

func TestTTY_AnimatesOnlyLiveStagesAndTransitionsToProgress(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter, backend := ttyTestReporter(t, true, false)

	g.Expect(reporter.Begin("Processing", 2)).To(Succeed())
	summary, err := reporter.StartStage("Preparing", StageSummary, WorkNone)
	g.Expect(err).NotTo(HaveOccurred())

	if summary == nil {
		panic("StartStage returned a nil stage without an error")
	}

	g.Expect(backend.started).To(BeEmpty())
	g.Expect(summary.Complete()).To(Succeed())

	live, err := reporter.StartStage("Acquiring", StageLive, WorkObservations)
	g.Expect(err).NotTo(HaveOccurred())

	if live == nil {
		panic("StartStage returned a nil stage without an error")
	}

	g.Expect(backend.started).To(HaveLen(1))
	g.Expect(backend.started[0].kind).To(Equal("spinner"))

	g.Expect(live.SetStatus("scanning")).To(Succeed())
	g.Expect(live.SetTotal(20)).To(Succeed())
	g.Expect(backend.started).To(HaveLen(2))
	g.Expect(backend.started[0].stops).To(Equal(1))
	g.Expect(backend.started[1].kind).To(Equal("progress"))
	g.Expect(live.SetProgress(7)).To(Succeed())
	g.Expect(backend.started[1].current).To(Equal(int64(7)))
	g.Expect(live.Complete()).To(Succeed())
	g.Expect(backend.started[1].stops).To(Equal(1))
	g.Expect(strings.Join(backend.lines, "\n")).To(ContainSubstring("✓ Acquiring"))
}

func TestTTY_UsesASCIISafeSymbolsWhenUnicodeUnavailable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	reporter, backend := ttyTestReporter(t, false, true)

	g.Expect(reporter.Begin("Processing", 1)).To(Succeed())
	stage, err := reporter.StartStage("Preparing", StageSummary, WorkNone)
	g.Expect(err).NotTo(HaveOccurred())

	if stage == nil {
		panic("StartStage returned a nil stage without an error")
	}

	g.Expect(stage.Complete()).To(Succeed())

	lines := strings.Join(backend.lines, "\n")
	g.Expect(lines).To(ContainSubstring("[OK] Preparing"))
	g.Expect(lines).NotTo(ContainSubstring("\x1b"))
}

func TestTTY_OutcomesAndCloseStopActiveDisplay(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		outcome func(Stage) error
		want    string
	}{
		{name: "failure", outcome: func(s Stage) error { return s.Fail(errors.New("boom")) }, want: "failed"},
		{name: "cancellation", outcome: func(s Stage) error { return s.Cancel(errors.New("stop")) }, want: "cancelled"},
		{name: "early close", outcome: nil, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			reporter, backend := ttyTestReporter(t, true, false)
			g.Expect(reporter.Begin("Processing", 1)).To(Succeed())
			stage, err := reporter.StartStage("Work", StageLive, WorkCommits)
			g.Expect(err).NotTo(HaveOccurred())

			if tc.outcome != nil {
				g.Expect(tc.outcome(stage)).To(Succeed())
			}

			g.Expect(reporter.Close()).To(Succeed())

			g.Expect(backend.started[0].stops).To(Equal(1))

			if tc.want != "" {
				g.Expect(strings.Join(backend.lines, "\n")).To(ContainSubstring(tc.want))
			}
		})
	}
}

func TestTTY_AutoFallsBackOnceButForcedTTYReturnsInitializationError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	initErr := errors.New("terminal init failed")
	factory := func(resolvedConfig) (renderer, error) { return nil, initErr }

	var autoOutput bytes.Buffer

	auto, err := newConfiguredReporter(Config{
		Mode:       ModeAuto,
		Writer:     &autoOutput,
		IsTerminal: func(io.Writer) bool { return true },
		LookupEnv:  env(nil),
	}, factory)
	g.Expect(err).NotTo(HaveOccurred())

	if auto == nil {
		panic("newConfiguredReporter returned nil without an error")
	}

	g.Expect(auto.Begin("Processing", 0)).To(Succeed())
	g.Expect(auto.Finish()).To(Succeed())
	g.Expect(strings.Count(autoOutput.String(), "falling back to plain progress")).To(Equal(1))
	g.Expect(autoOutput.String()).To(ContainSubstring("Processing"))

	_, err = newConfiguredReporter(Config{
		Mode:       ModeTTY,
		Writer:     io.Discard,
		IsTerminal: func(io.Writer) bool { return true },
		LookupEnv:  env(nil),
	}, factory)
	g.Expect(err).To(MatchError(initErr))
}
