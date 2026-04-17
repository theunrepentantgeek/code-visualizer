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
	Quiet   bool   `help:"Suppress progress output; show only the final result." short:"q"`
	Verbose bool   `help:"Show detailed progress during scanning and metric calculation." short:"v"`
	Debug   bool   `help:"Show per-directory scan progress (implies verbose output)."`
	Config  string `help:"Path to configuration file (.yaml, .yml, or .json)." name:"config" optional:""`

	//nolint:revive // Long help text is more important than minimizing line length, and annotations can't be wrapped
	ExportConfig string `help:"Write effective configuration to file (.yaml, .yml, or .json)." name:"export-config" optional:""`

	Render RenderCmd `cmd:"" help:"Render a visualization."`
}

func (c *CLI) Validate() error {
	count := 0
	if c.Quiet {
		count++
	}

	if c.Verbose {
		count++
	}

	if c.Debug {
		count++
	}

	if count > 1 {
		return errors.New("--quiet, --verbose, and --debug are mutually exclusive")
	}

	return nil
}

// Flags bundles cross-cutting concerns that are passed to every command's Run method.
type Flags struct {
	Quiet        bool
	Verbose      bool
	Debug        bool
	ExportConfig string
	Config       *config.Config
}

// logPhase logs an Info-level phase message unless quiet mode is active.
func (f *Flags) logPhase(msg string, args ...any) {
	if !f.Quiet {
		slog.Info(msg, args...)
	}
}

func setupLogger(verbose, debug bool) { //nolint:revive // flag-parameter: boolean toggles are idiomatic for log verbosity
	level := slog.LevelInfo
	if verbose || debug {
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
	setupLogger(false, false)

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

	setupLogger(cli.Verbose, cli.Debug)

	slog.Info("codeviz", "version", "dev")

	cfg := config.New()

	if cli.Config != "" {
		if loadErr := cfg.Load(cli.Config); loadErr != nil {
			exitWithError(loadErr, 5)
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
		exitWithError(err, code)
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

func exitWithError(err error, code int) {
	slog.Error(err.Error())
	os.Exit(code) //nolint:revive // deep-exit: intentional exit from CLI error handler called by main
}
