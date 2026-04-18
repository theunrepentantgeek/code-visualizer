package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/alecthomas/kong"
	"github.com/lmittmann/tint"

	"github.com/bevan/code-visualizer/internal/config"
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/provider/git"
)

type CLI struct {
	Quiet   bool   `help:"Suppress all non-essential output; only warnings and errors are shown." short:"q" xor:"verbosity"` //nolint:revive // kong struct tags require long lines
	Verbose bool   `help:"Show detailed progress during scanning and metric calculation." short:"v" xor:"verbosity"`
	Debug   bool   `help:"Show per-directory scan progress (implies verbose output)." xor:"verbosity"`
	Config  string `help:"Path to configuration file (.yaml, .yml, or .json)." name:"config" optional:""`

	//nolint:revive // Long help text is more important than minimizing line length, and annotations can't be wrapped
	ExportConfig string `help:"Write effective configuration to file (.yaml, .yml, or .json)." name:"export-config" optional:""`

	Render RenderCmd `cmd:"" help:"Render a visualization."`
}

// Flags bundles cross-cutting concerns that are passed to every command's Run method.
type Flags struct {
	Quiet        bool
	Verbose      bool
	Debug        bool
	ExportConfig string
	Config       *config.Config
}

func setupLogger(quiet, verbose, debug bool) { //nolint:revive // flag-parameter: boolean toggles are idiomatic for log verbosity
	level := slog.LevelInfo

	if quiet {
		level = slog.LevelWarn
	} else if verbose || debug {
		level = slog.LevelDebug
	}

	noColor := os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"

	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:   level,
		NoColor: noColor,
	})
	slog.SetDefault(slog.New(handler))
}

func countAll(node *model.Directory) (files int, dirs int) {
	files = len(node.Files)
	for _, d := range node.Dirs {
		dirs++
		f, d2 := countAll(d)
		files += f
		dirs += d2
	}

	return files, dirs
}

func main() {
	filesystem.Register()
	git.Register()

	// Install tint early so bootstrap errors are formatted consistently.
	setupLogger(false, false, false)

	cli := CLI{}

	parser, err := kong.New(&cli,
		kong.Name("codeviz"),
		kong.Description("Generate visualizations of file trees."),
	)
	if err != nil {
		slog.Error("failed to initialize CLI", "error", err)
		os.Exit(5)
	}

	ctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		var parseErr *kong.ParseError
		if errors.As(err, &parseErr) && parseErr.Context != nil {
			_ = parseErr.Context.PrintUsage(false)
		}

		slog.Error(err.Error())
		os.Exit(1)
	}

	setupLogger(cli.Quiet, cli.Verbose, cli.Debug)

	slog.Info("codeviz", "version", "dev")

	cfg := config.New()

	if cli.Config != "" {
		if loadErr := cfg.Load(cli.Config); loadErr != nil {
			slog.Error(loadErr.Error())
			os.Exit(5)
		}
	}

	flags := &Flags{
		Quiet:        cli.Quiet,
		Verbose:      cli.Verbose,
		Debug:        cli.Debug,
		ExportConfig: cli.ExportConfig,
		Config:       cfg,
	}

	err = ctx.Run(flags)
	if err != nil {
		code := classifyError(err)
		slog.Error(err.Error())
		os.Exit(code)
	}
}

func classifyError(err error) int {
	var (
		gitErr     *gitRequiredError
		targetErr  *targetPathError
		outputErr  *outputPathError
		noFilesErr *noFilesAfterFilterError
	)

	switch {
	case errors.As(err, &targetErr):
		return 2
	case errors.As(err, &gitErr):
		return 3
	case errors.As(err, &outputErr):
		return 4
	case errors.As(err, &noFilesErr):
		return 6
	default:
		return 5
	}
}

type gitRequiredError struct {
	metric metric.Name
	target string
}

func (e *gitRequiredError) Error() string {
	return fmt.Sprintf("metric %q requires a git repository, but %q is not a git repository", e.metric, e.target)
}

type targetPathError struct {
	msg string
}

func (e *targetPathError) Error() string { return e.msg }

type outputPathError struct {
	msg string
}

func (e *outputPathError) Error() string { return e.msg }

type noFilesAfterFilterError struct {
	msg string
}

func (e *noFilesAfterFilterError) Error() string { return e.msg }
