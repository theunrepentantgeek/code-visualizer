package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/alecthomas/kong"

	"github.com/bevan/code-visualizer/internal/config"
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/scan"
)

type CLI struct {
	Verbose      bool   `help:"Enable debug-level logging." short:"v"`
	Format       string `default:"text" enum:"text,json" help:"Diagnostic/error output format (text, json)."`
	Config       string `help:"Path to configuration file (.yaml, .yml, or .json)." name:"config" optional:""`
	ExportConfig string `help:"Write effective configuration to file (.yaml, .yml, or .json)." name:"export-config" optional:""`

	Render RenderCmd `cmd:"" help:"Render a visualization."`
}

// Flags bundles cross-cutting concerns that are passed to every command's Run method.
type Flags struct {
	Verbose      bool
	Format       string
	ExportConfig string
	Config       *config.Config
}

func setupLogger(verbose bool) { //nolint:revive // flag-parameter: boolean toggle is idiomatic for log verbosity
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

func countAll(node scan.DirectoryNode) (files int, dirs int) {
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
	cli := CLI{}

	parser, err := kong.New(&cli,
		kong.Name("codeviz"),
		kong.Description("Generate visualizations of file trees."),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(5)
	}

	ctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		// Kong parse/validation errors are argument failures → show help, then exit 1
		var parseErr *kong.ParseError
		if errors.As(err, &parseErr) && parseErr.Context != nil {
			_ = parseErr.Context.PrintUsage(false)
		}

		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	setupLogger(cli.Verbose)

	cfg := config.New()

	if cli.Config != "" {
		if err := config.Load(cli.Config, cfg); err != nil {
			exitWithError(cli.Format, err, 5)
		}
	}

	flags := &Flags{
		Verbose:      cli.Verbose,
		Format:       cli.Format,
		ExportConfig: cli.ExportConfig,
		Config:       cfg,
	}

	err = ctx.Run(flags)
	if err != nil {
		code := classifyError(err)
		exitWithError(cli.Format, err, code)
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
	metric metric.MetricName
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

func exitWithError(format string, err error, code int) {
	if format == "json" {
		out := struct {
			Error string `json:"error"`
			Code  int    `json:"code"`
		}{
			Error: err.Error(),
			Code:  code,
		}

		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")

		if encErr := enc.Encode(out); encErr != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		}
	} else {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
	}

	os.Exit(code) //nolint:revive // deep-exit: intentional exit from CLI error handler called by main
}
