// Package progress reports command workflow progress independently of the
// processing operations that produce it.
package progress //nolint:revive // The package intentionally exposes its small configuration and event model.

import (
	"io"
	"time"
)

type Mode string

const (
	ModeAuto  Mode = "auto"
	ModeTTY   Mode = "tty"
	ModePlain Mode = "plain"
)

type StageKind uint8

const (
	StageSummary StageKind = iota
	StageLive
)

type WorkKind uint8

const (
	WorkNone WorkKind = iota
	WorkObservations
	WorkCommits
)

type Config struct {
	Mode            Mode
	Writer          io.Writer
	IsTerminal      func(io.Writer) bool
	SupportsUnicode func() bool
	LookupEnv       func(string) (string, bool)
	NoColor         bool
	Quiet           bool
	Verbose         bool
	Now             func() time.Time
}

type Sink interface {
	WorkKind() WorkKind
	SetTotal(total int64) error
	SetProgress(current int64) error
	SetStatus(message string) error
}

type Stage interface {
	Sink
	Complete() error
	Fail(err error) error
	Cancel(err error) error
}

type Reporter interface {
	Begin(title string, stageCount int) error
	StartStage(name string, kind StageKind, work WorkKind) (Stage, error)
	Finish() error
	DiagnosticWriter() io.Writer
	Close() error
}
