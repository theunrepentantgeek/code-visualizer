package progress

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/pterm/pterm"
)

type liveDisplay interface {
	updateText(text string) error
	setCurrent(current int64) error
	stop() error
}

type ttyBackend interface {
	startSpinner(text string, ascii bool) (liveDisplay, error)
	startProgress(text string, total int64, ascii bool) (liveDisplay, error)
	println(text string) error
}

type ttyRenderer struct {
	config      resolvedConfig
	backend     ttyBackend
	active      liveDisplay
	activeLabel string
	total       int64
	current     int64
}

func newTTYRenderer(config resolvedConfig, backend ttyBackend) (renderer, error) {
	if backend == nil {
		return nil, errors.New("TTY backend is required")
	}

	return &ttyRenderer{config: config, backend: backend}, nil
}

func newDefaultTTYRenderer(config resolvedConfig) (renderer, error) {
	return newTTYRenderer(config, &ptermBackend{
		writer:  config.Writer,
		noColor: config.noColor,
	})
}

//nolint:cyclop,revive // Event rendering is intentionally an exhaustive lifecycle switch.
func (r *ttyRenderer) render(e event) error {
	switch e.kind {
	case eventBegun:
		return r.backend.println(cleanLine(e.title))
	case eventStageStarted:
		if e.stageKind == StageLive {
			r.activeLabel = cleanLine(e.name)

			display, err := r.backend.startSpinner(r.activeLabel, !r.supportsUnicode())
			if err != nil {
				return err
			}

			r.active = display
			r.total = 0
			r.current = 0
		}
	case eventProgress:
		return r.renderProgress(e)
	case eventStatus:
		if r.active != nil {
			r.activeLabel = cleanLine(e.name) + ": " + cleanLine(e.status)

			return r.active.updateText(r.activeLabel)
		}
	case eventStageSucceeded, eventStageFailed, eventStageCancelled:
		if err := r.stopActive(); err != nil {
			return err
		}

		err := r.renderOutcome(e)
		r.activeLabel = ""
		r.total = 0
		r.current = 0

		return err
	case eventFinished:
		return r.backend.println(r.decorate(r.successSymbol(), cleanLine(e.title)+": done"))
	default:
		return nil
	}

	return nil
}

func (r *ttyRenderer) renderProgress(e event) error {
	if e.total > 0 && r.total == 0 {
		if err := r.stopActive(); err != nil {
			return err
		}

		display, err := r.backend.startProgress(cleanLine(e.name), e.total, !r.supportsUnicode())
		if err != nil {
			return err
		}

		r.active = display
		r.total = e.total
	}

	if r.active != nil && r.total > 0 {
		if err := r.active.setCurrent(e.current); err != nil {
			return err
		}

		r.current = e.current
	}

	return nil
}

func (r *ttyRenderer) renderOutcome(e event) error {
	message := fmt.Sprintf("%s (%s)", cleanLine(e.name), formatElapsed(e.elapsed))

	switch e.kind {
	case eventStageSucceeded:
		return r.backend.println(r.decorate(r.successSymbol(), message))
	case eventStageFailed:
		return r.backend.println(r.decorate(r.failureSymbol(), message+": failed: "+cleanError(e.err)))
	case eventStageCancelled:
		return r.backend.println(r.decorate(r.cancelSymbol(), message+": cancelled: "+cleanError(e.err)))
	default:
		return nil
	}
}

func (r *ttyRenderer) decorate(symbol, message string) string {
	line := symbol + " " + message
	if r.config.noColor {
		return pterm.RemoveColorFromString(line)
	}

	return line
}

func (r *ttyRenderer) successSymbol() string {
	if r.supportsUnicode() {
		return "✓"
	}

	return "[OK]"
}

func (r *ttyRenderer) failureSymbol() string {
	if r.supportsUnicode() {
		return "✗"
	}

	return "[FAIL]"
}

func (r *ttyRenderer) cancelSymbol() string {
	if r.supportsUnicode() {
		return "■"
	}

	return "[CANCEL]"
}

func (r *ttyRenderer) supportsUnicode() bool {
	return r.config.SupportsUnicode == nil || r.config.SupportsUnicode()
}

func (r *ttyRenderer) stopActive() error {
	if r.active == nil {
		return nil
	}

	err := r.active.stop()
	r.active = nil

	return err
}

func (r *ttyRenderer) writeDiagnostic(line string) error {
	wasActive := r.active != nil
	if err := r.stopActive(); err != nil {
		return err
	}

	if err := r.backend.println(strings.TrimSuffix(line, "\r")); err != nil {
		return err
	}

	if !wasActive {
		return nil
	}

	if r.total > 0 {
		display, err := r.backend.startProgress(r.activeLabel, r.total, !r.supportsUnicode())
		if err != nil {
			return err
		}

		r.active = display

		return r.active.setCurrent(r.current)
	}

	display, err := r.backend.startSpinner(r.activeLabel, !r.supportsUnicode())
	if err != nil {
		return err
	}

	r.active = display

	return nil
}

func (r *ttyRenderer) close() error { return r.stopActive() }

type ptermBackend struct {
	writer  io.Writer
	noColor bool
}

//nolint:revive // ASCII selects terminal-safe PTerm glyphs.
func (b *ptermBackend) startSpinner(text string, ascii bool) (liveDisplay, error) {
	printer := pterm.DefaultSpinner.WithWriter(b.writer).WithRemoveWhenDone(true).WithShowTimer(false)
	if ascii {
		printer = printer.WithSequence("-", "\\", "|", "/")
	}

	if b.noColor {
		printer = printer.WithStyle(pterm.NewStyle()).WithMessageStyle(pterm.NewStyle())
	}

	started, err := printer.Start(text)
	if err != nil {
		return nil, fmt.Errorf("start terminal spinner: %w", err)
	}

	return &ptermSpinner{printer: started}, nil
}

//nolint:revive // ASCII selects terminal-safe PTerm glyphs.
func (b *ptermBackend) startProgress(text string, total int64, ascii bool) (liveDisplay, error) {
	if total > int64(math.MaxInt) {
		return nil, fmt.Errorf("progress total %d exceeds terminal renderer capacity", total)
	}

	printer := pterm.DefaultProgressbar.
		WithWriter(b.writer).
		WithTotal(int(total)).
		WithTitle(text).
		WithShowCount(true).
		WithShowPercentage(true).
		WithRemoveWhenDone(true)
	if ascii {
		printer = printer.WithBarCharacter("=").WithBarFiller(" ").WithLastCharacter(">")
	}

	if b.noColor {
		printer = printer.WithBarStyle(pterm.NewStyle()).WithTitleStyle(pterm.NewStyle())
	}

	started, err := printer.Start()
	if err != nil {
		return nil, fmt.Errorf("start terminal progress: %w", err)
	}

	return &ptermProgress{printer: started}, nil
}

func (b *ptermBackend) println(text string) error {
	_, err := fmt.Fprintln(b.writer, text)
	if err != nil {
		return fmt.Errorf("write terminal progress: %w", err)
	}

	return nil
}

type ptermSpinner struct {
	printer *pterm.SpinnerPrinter
}

func (d *ptermSpinner) updateText(text string) error {
	d.printer.UpdateText(text)

	return nil
}

func (*ptermSpinner) setCurrent(int64) error { return nil }
func (d *ptermSpinner) stop() error {
	if err := d.printer.Stop(); err != nil {
		return fmt.Errorf("stop terminal spinner: %w", err)
	}

	return nil
}

type ptermProgress struct {
	printer *pterm.ProgressbarPrinter
}

func (d *ptermProgress) updateText(text string) error {
	d.printer.UpdateTitle(text)

	return nil
}

func (d *ptermProgress) setCurrent(current int64) error {
	d.printer.Add(int(current) - d.printer.Current)

	return nil
}

func (d *ptermProgress) stop() error {
	_, err := d.printer.Stop()
	if err != nil {
		return fmt.Errorf("stop terminal progress: %w", err)
	}

	return nil
}
