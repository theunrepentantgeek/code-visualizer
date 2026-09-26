package progress

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type plainRenderer struct {
	writer         io.Writer
	now            func() time.Time
	lastProgressAt time.Time
	lastCurrent    int64
	lastPercentage int64
}

func newPlainRenderer(config resolvedConfig) renderer {
	return &plainRenderer{
		writer:         config.Writer,
		now:            config.Now,
		lastPercentage: -1,
	}
}

func (r *plainRenderer) render(e event) error {
	switch e.kind {
	case eventBegun:
		return r.writeLine("%s", cleanLine(e.title))
	case eventStageStarted:
		r.lastProgressAt = r.now()
		r.lastCurrent = 0
		r.lastPercentage = 0

		return r.writeLine("%s: started", r.prefix(e))
	case eventProgress:
		return r.renderProgress(e)
	case eventStatus:
		return r.writeLine("%s: %s", r.prefix(e), cleanLine(e.status))
	case eventStageSucceeded:
		return r.writeLine("%s: done (%s)", r.prefix(e), formatElapsed(e.elapsed))
	case eventStageFailed:
		return r.writeLine("%s: failed (%s): %s", r.prefix(e), formatElapsed(e.elapsed), cleanError(e.err))
	case eventStageCancelled:
		return r.writeLine("%s: cancelled (%s): %s", r.prefix(e), formatElapsed(e.elapsed), cleanError(e.err))
	case eventFinished:
		return r.writeLine("%s: done", cleanLine(e.title))
	default:
		return nil
	}
}

func (r *plainRenderer) renderProgress(e event) error {
	if e.total <= 0 || e.current == r.lastCurrent {
		return nil
	}

	percentage := min(e.current*100/e.total, 100)
	now := r.now()
	percentageAdvanced := percentage-r.lastPercentage >= 5
	timeElapsed := now.Sub(r.lastProgressAt) >= 10*time.Second

	finished := e.current == e.total
	if !percentageAdvanced && !timeElapsed && !finished {
		return nil
	}

	if err := r.writeLine(
		"%s: %d/%d (%d%%)",
		r.prefix(e),
		e.current,
		e.total,
		percentage,
	); err != nil {
		return err
	}

	r.lastCurrent = e.current
	r.lastPercentage = percentage
	r.lastProgressAt = now

	return nil
}

func (*plainRenderer) prefix(e event) string {
	return fmt.Sprintf("[%d/%d] %s", e.stageIndex, e.stageCount, cleanLine(e.name))
}

func (r *plainRenderer) writeLine(format string, args ...any) error {
	_, err := fmt.Fprintf(r.writer, format+"\n", args...)
	if err != nil {
		return fmt.Errorf("write plain progress: %w", err)
	}

	return nil
}

func (r *plainRenderer) writeDiagnostic(line string) error {
	return r.writeLine("%s", strings.TrimSuffix(line, "\r"))
}
func (*plainRenderer) close() error { return nil }

func cleanError(err error) string {
	if err == nil {
		return ""
	}

	return cleanLine(err.Error())
}

func cleanLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func formatElapsed(elapsed time.Duration) string {
	return fmt.Sprintf("%.1fs", elapsed.Seconds())
}
