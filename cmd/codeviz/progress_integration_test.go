package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

//nolint:paralleltest // runApplication replaces the process-wide default logger.
func TestRunApplication_PlainAndRedirectedAutoUsePlainProgress(t *testing.T) {
	g := NewWithT(t)
	target := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(target, "main.go"), []byte("package main\n"), 0o600)).To(Succeed())

	for _, mode := range []string{"plain", "auto"} {
		var stdout, stderr bytes.Buffer

		code := runApplication(application{
			args: []string{
				"--progress=" + mode, "tree-map", target,
				"-o", filepath.Join(target, mode+".png"),
				"-s", "file-size",
			},
			stdout:     &stdout,
			stderr:     &stderr,
			context:    context.Background(),
			isTerminal: func(io.Writer) bool { return false },
			lookupEnv:  func(string) (string, bool) { return "", false },
		})

		g.Expect(code).To(Equal(0), stderr.String())
		g.Expect(stdout.String()).To(BeEmpty())
		g.Expect(stderr.String()).To(ContainSubstring("Tree map"))
		g.Expect(stderr.String()).To(ContainSubstring("[2/4] Acquiring data"))
		g.Expect(stderr.String()).NotTo(ContainSubstring("\x1b"))
	}
}

//nolint:paralleltest // runApplication replaces the process-wide default logger.
func TestRunApplication_PreCancelledContextReturns130(t *testing.T) {
	g := NewWithT(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	target := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(target, "main.go"), []byte("package main\n"), 0o600)).To(Succeed())

	var stderr bytes.Buffer

	code := runApplication(application{
		args: []string{
			"--progress=plain", "tree-map", target,
			"-o", filepath.Join(target, "out.png"),
			"-s", "file-size",
		},
		stdout:     &bytes.Buffer{},
		stderr:     &stderr,
		context:    ctx,
		isTerminal: func(io.Writer) bool { return false },
		lookupEnv:  func(string) (string, bool) { return "", false },
	})

	g.Expect(code).To(Equal(130), stderr.String())
	g.Expect(stderr.String()).To(ContainSubstring("cancelled"))
	g.Expect(strings.ToLower(stderr.String())).NotTo(ContainSubstring("succeeded"))
	g.Expect(stderr.String()).NotTo(ContainSubstring("Rendering"))
	g.Expect(stderr.String()).NotTo(ContainSubstring("Writing output"))
}

//nolint:paralleltest // runApplication replaces the process-wide default logger.
func TestRunApplication_QuietSuppressesProgress(t *testing.T) {
	g := NewWithT(t)
	target := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(target, "main.go"), []byte("package main\n"), 0o600)).To(Succeed())

	var stderr bytes.Buffer

	code := runApplication(application{
		args: []string{
			"--quiet", "tree-map", target,
			"-o", filepath.Join(target, "out.png"),
			"-s", "file-size",
		},
		stdout:     &bytes.Buffer{},
		stderr:     &stderr,
		context:    context.Background(),
		isTerminal: func(io.Writer) bool { return false },
		lookupEnv:  func(string) (string, bool) { return "", false },
	})

	g.Expect(code).To(Equal(0), stderr.String())
	g.Expect(stderr.String()).To(BeEmpty())
}

//nolint:paralleltest // runApplication replaces the process-wide default logger.
func TestRunApplication_ForcedTTYRequiresTerminal(t *testing.T) {
	g := NewWithT(t)

	var stderr bytes.Buffer

	code := runApplication(application{
		args:       []string{"--progress=tty", "tree-map", ".", "-o", "out.png", "-s", "file-size"},
		stdout:     &bytes.Buffer{},
		stderr:     &stderr,
		context:    context.Background(),
		isTerminal: func(io.Writer) bool { return false },
		lookupEnv:  func(string) (string, bool) { return "", false },
	})

	g.Expect(code).To(Equal(5), stderr.String())
	g.Expect(strings.ToLower(stderr.String())).To(ContainSubstring("terminal"))
}

//nolint:paralleltest // runApplication replaces the process-wide default logger.
func TestRunApplication_InvalidProgressModeIsUsageFailure(t *testing.T) {
	g := NewWithT(t)

	var stderr bytes.Buffer

	code := runApplication(application{
		args:       []string{"--progress=animated", "tree-map", ".", "-o", "out.png"},
		stdout:     &bytes.Buffer{},
		stderr:     &stderr,
		context:    context.Background(),
		isTerminal: func(io.Writer) bool { return false },
		lookupEnv:  func(string) (string, bool) { return "", false },
	})

	g.Expect(code).To(Equal(1), stderr.String())
	g.Expect(stderr.String()).To(ContainSubstring("must be one of"))
	g.Expect(stderr.String()).To(ContainSubstring("animated"))
}

//nolint:paralleltest // runApplication replaces the process-wide default logger.
func TestRunApplication_HelpUsesStdoutWithoutProgress(t *testing.T) {
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	code := runApplication(application{
		args:       []string{"help", "metrics"},
		stdout:     &stdout,
		stderr:     &stderr,
		context:    context.Background(),
		isTerminal: func(io.Writer) bool { return false },
		lookupEnv:  func(string) (string, bool) { return "", false },
	})

	g.Expect(code).To(Equal(0), stderr.String())
	g.Expect(stdout.String()).To(ContainSubstring("file-size"))
	g.Expect(stderr.String()).To(BeEmpty())
}

//nolint:paralleltest // runApplication replaces the process-wide default logger.
func TestRunApplication_NoColorKeepsPlainProgressANSIFree(t *testing.T) {
	g := NewWithT(t)
	target := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(target, "main.go"), []byte("package main\n"), 0o600)).To(Succeed())

	var stderr bytes.Buffer

	code := runApplication(application{
		args: []string{
			"--progress=plain", "--no-color", "tree-map", target,
			"-o", filepath.Join(target, "out.png"),
			"-s", "file-size",
		},
		stdout:     &bytes.Buffer{},
		stderr:     &stderr,
		context:    context.Background(),
		isTerminal: func(io.Writer) bool { return false },
		lookupEnv:  func(string) (string, bool) { return "", false },
	})

	g.Expect(code).To(Equal(0), stderr.String())
	g.Expect(stderr.String()).NotTo(ContainSubstring("\x1b"))
}

//nolint:paralleltest // runApplication replaces the process-wide default logger.
func TestRunApplication_ProcessingFailureLeavesDurableFailedStage(t *testing.T) {
	g := NewWithT(t)
	target := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(target, "main.go"), []byte("package main\n"), 0o600)).To(Succeed())
	outputDirectory := filepath.Join(target, "out.png")
	g.Expect(os.Mkdir(outputDirectory, 0o700)).To(Succeed())

	var stderr bytes.Buffer

	code := runApplication(application{
		args: []string{
			"--progress=plain", "tree-map", target,
			"-o", outputDirectory,
			"-s", "file-size",
		},
		stdout:     &bytes.Buffer{},
		stderr:     &stderr,
		context:    context.Background(),
		isTerminal: func(io.Writer) bool { return false },
		lookupEnv:  func(string) (string, bool) { return "", false },
	})

	g.Expect(code).To(Equal(5), stderr.String())
	g.Expect(stderr.String()).To(ContainSubstring("Writing output: failed"))
}

func TestEnvironmentSupportsUnicode_UsesLocale(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	for _, locale := range []string{"en_US.UTF-8", "C.utf8"} {
		g.Expect(environmentSupportsUnicode(func(key string) (string, bool) {
			if key == "LANG" {
				return locale, true
			}

			return "", false
		})).To(BeTrue(), locale)
	}

	g.Expect(environmentSupportsUnicode(func(string) (string, bool) {
		return "", false
	})).To(BeFalse())
}
