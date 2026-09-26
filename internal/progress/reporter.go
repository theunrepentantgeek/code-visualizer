package progress

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

type eventKind uint8

const (
	eventBegun eventKind = iota
	eventStageStarted
	eventProgress
	eventStatus
	eventStageSucceeded
	eventStageFailed
	eventStageCancelled
	eventFinished
)

type event struct {
	kind       eventKind
	title      string
	stageCount int
	stageIndex int
	name       string
	stageKind  StageKind
	workKind   WorkKind
	total      int64
	current    int64
	status     string
	err        error
	elapsed    time.Duration
}

type renderer interface {
	render(event event) error
	writeDiagnostic(line string) error
	close() error
}

type discardRenderer struct {
	writer io.Writer
}

func (*discardRenderer) render(event) error { return nil }

func (r *discardRenderer) writeDiagnostic(line string) error {
	_, err := fmt.Fprintln(r.writer, line)
	if err != nil {
		return fmt.Errorf("write discarded progress diagnostic: %w", err)
	}

	return nil
}
func (*discardRenderer) close() error { return nil }

type reporter struct {
	mu         sync.Mutex
	config     Config
	renderer   renderer
	begun      bool
	finished   bool
	closed     bool
	title      string
	stageCount int
	completed  int
	active     *stage
	diagnostic *diagnosticWriter
}

type stage struct {
	reporter *reporter
	index    int
	name     string
	kind     StageKind
	work     WorkKind
	started  time.Time
	total    int64
	current  int64
	finished bool
}

func New(config Config) (Reporter, error) {
	return newConfiguredReporter(config, newDefaultTTYRenderer)
}

type ttyRendererFactory func(resolvedConfig) (renderer, error)

//nolint:cyclop,revive // Renderer selection is clearer as one guarded composition function.
func newConfiguredReporter(config Config, ttyFactory ttyRendererFactory) (Reporter, error) {
	if config.Writer == nil {
		return nil, errors.New("progress writer is required")
	}

	if config.Now == nil {
		config.Now = time.Now
	}

	if config.LookupEnv == nil {
		config.LookupEnv = os.LookupEnv
	}

	if config.IsTerminal == nil {
		config.IsTerminal = func(writer io.Writer) bool {
			file, ok := writer.(interface{ Fd() uintptr })

			return ok && term.IsTerminal(int(file.Fd()))
		}
	}

	resolved, err := resolveConfig(config)
	if err != nil {
		return nil, err
	}

	var selected renderer

	switch {
	case config.Quiet:
		selected = &discardRenderer{writer: config.Writer}
	case resolved.mode == ModePlain:
		selected = newPlainRenderer(resolved)
	default:
		selected, err = ttyFactory(resolved)
		if err != nil {
			requested, parseErr := ParseMode(string(config.Mode))
			if parseErr != nil {
				return nil, parseErr
			}

			if requested == ModeTTY {
				return nil, err
			}

			if _, writeErr := fmt.Fprintf(
				config.Writer,
				"terminal progress unavailable; falling back to plain progress: %s\n",
				cleanError(err),
			); writeErr != nil {
				return nil, errors.Join(err, writeErr)
			}

			selected = newPlainRenderer(resolved)
		}
	}

	return newReporter(config, selected), nil
}

func newReporter(config Config, renderer renderer) *reporter {
	if config.Now == nil {
		config.Now = time.Now
	}

	return &reporter{config: config, renderer: renderer}
}

func (r *reporter) Begin(title string, stageCount int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return errors.New("progress reporter is closed")
	}

	if r.begun {
		return errors.New("progress reporter already begun")
	}

	if stageCount < 0 {
		return errors.New("progress stage count cannot be negative")
	}

	e := event{kind: eventBegun, title: title, stageCount: stageCount}
	if err := r.renderer.render(e); err != nil {
		return err
	}

	r.begun = true
	r.title = title
	r.stageCount = stageCount

	return nil
}

func (r *reporter) StartStage(name string, kind StageKind, work WorkKind) (Stage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.begun {
		return nil, errors.New("progress reporter must begin before starting a stage")
	}

	if r.finished {
		return nil, errors.New("progress reporter already finished")
	}

	if r.active != nil {
		return nil, fmt.Errorf("progress stage %q is already active", r.active.name)
	}

	if r.completed >= r.stageCount {
		return nil, fmt.Errorf("all %d progress stages are already complete", r.stageCount)
	}

	if kind == StageSummary && work != WorkNone {
		return nil, errors.New("summary progress stage cannot have a work kind")
	}

	next := &stage{
		reporter: r,
		index:    r.completed + 1,
		name:     name,
		kind:     kind,
		work:     work,
		started:  r.config.Now(),
	}
	if err := r.renderer.render(next.snapshot(eventStageStarted)); err != nil {
		return nil, err
	}

	r.active = next

	return next, nil
}

func (r *reporter) Finish() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.begun {
		return errors.New("progress reporter has not begun")
	}

	if r.finished {
		return errors.New("progress reporter already finished")
	}

	if r.active != nil {
		return fmt.Errorf("progress stage %q is still active", r.active.name)
	}

	if r.completed != r.stageCount {
		return fmt.Errorf("only %d of %d progress stages completed", r.completed, r.stageCount)
	}

	if err := r.renderer.render(event{
		kind:       eventFinished,
		title:      r.title,
		stageCount: r.stageCount,
	}); err != nil {
		return err
	}

	r.finished = true

	return nil
}

func (r *reporter) DiagnosticWriter() io.Writer {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.diagnostic == nil {
		r.diagnostic = &diagnosticWriter{reporter: r}
	}

	return r.diagnostic
}

func (r *reporter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	err := r.diagnostic.flushLocked()
	err = errors.Join(err, r.renderer.close())
	r.closed = true

	return err
}

func (s *stage) WorkKind() WorkKind { return s.work }

func (s *stage) SetTotal(total int64) error {
	s.reporter.mu.Lock()
	defer s.reporter.mu.Unlock()

	if err := s.validateActive(); err != nil {
		return err
	}

	if s.kind != StageLive {
		return errors.New("summary progress stage cannot have a total")
	}

	if total <= 0 {
		return errors.New("progress total must be positive")
	}

	if s.total != 0 {
		return errors.New("progress total is already set")
	}

	e := s.snapshot(eventProgress)

	e.total = total
	if err := s.reporter.renderer.render(e); err != nil {
		return err
	}

	s.total = total

	return nil
}

func (s *stage) SetProgress(current int64) error {
	s.reporter.mu.Lock()
	defer s.reporter.mu.Unlock()

	if err := s.validateActive(); err != nil {
		return err
	}

	if s.kind != StageLive {
		return errors.New("summary progress stage cannot report progress")
	}

	if current < 0 {
		return errors.New("progress current value cannot be negative")
	}

	if current < s.current {
		return errors.New("progress current value cannot decrease")
	}

	if s.total > 0 && current > s.total {
		return errors.New("progress current value cannot exceed total")
	}

	e := s.snapshot(eventProgress)

	e.current = current
	if err := s.reporter.renderer.render(e); err != nil {
		return err
	}

	s.current = current

	return nil
}

func (s *stage) SetStatus(message string) error {
	s.reporter.mu.Lock()
	defer s.reporter.mu.Unlock()

	if err := s.validateActive(); err != nil {
		return err
	}

	if !s.reporter.config.Verbose {
		return nil
	}

	e := s.snapshot(eventStatus)
	e.status = message

	return s.reporter.renderer.render(e)
}

func (s *stage) Complete() error {
	return s.finish(eventStageSucceeded, nil)
}

func (s *stage) Fail(err error) error {
	if err == nil {
		return errors.New("progress stage failure requires an error")
	}

	return s.finish(eventStageFailed, err)
}

func (s *stage) Cancel(err error) error {
	if err == nil {
		return errors.New("progress stage cancellation requires an error")
	}

	return s.finish(eventStageCancelled, err)
}

func (s *stage) finish(kind eventKind, stageErr error) error {
	s.reporter.mu.Lock()
	defer s.reporter.mu.Unlock()

	if err := s.validateActive(); err != nil {
		return err
	}

	if kind == eventStageSucceeded && s.total > 0 && s.current < s.total {
		progressEvent := s.snapshot(eventProgress)

		progressEvent.current = s.total
		if err := s.reporter.renderer.render(progressEvent); err != nil {
			return err
		}

		s.current = s.total
	}

	e := s.snapshot(kind)
	e.err = stageErr

	e.elapsed = s.reporter.config.Now().Sub(s.started)
	if err := s.reporter.renderer.render(e); err != nil {
		return err
	}

	s.finished = true
	s.reporter.active = nil
	s.reporter.completed++

	return nil
}

func (s *stage) validateActive() error {
	if s.finished {
		return fmt.Errorf("progress stage %q is already finished", s.name)
	}

	if s.reporter.active != s {
		return fmt.Errorf("progress stage %q is not active", s.name)
	}

	return nil
}

func (s *stage) snapshot(kind eventKind) event {
	return event{
		kind:       kind,
		stageCount: s.reporter.stageCount,
		stageIndex: s.index,
		name:       s.name,
		stageKind:  s.kind,
		workKind:   s.work,
		total:      s.total,
		current:    s.current,
	}
}
