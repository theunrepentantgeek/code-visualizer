//go:build unix

package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

//nolint:paralleltest,revive // The helper must terminate with the application exit code.
func TestProgressSignalHelper(_ *testing.T) {
	if os.Getenv("CODEVIZ_SIGNAL_HELPER") != "1" {
		return
	}

	runContext, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := runApplication(application{
		args: []string{
			"--progress=plain", "tree-map", os.Getenv("CODEVIZ_SIGNAL_TARGET"),
			"-o", os.Getenv("CODEVIZ_SIGNAL_OUTPUT"),
			"-s", "file-size",
		},
		stdout:     io.Discard,
		stderr:     os.Stderr,
		context:    runContext,
		isTerminal: func(io.Writer) bool { return false },
		lookupEnv:  os.LookupEnv,
	})

	stop()
	os.Exit(code)
}

//nolint:paralleltest,revive // This test owns process signals and a subprocess.
func TestCLI_SIGINTCancelsActiveProgressAndExits130(t *testing.T) {
	g := NewWithT(t)

	target := t.TempDir()
	for index := range 20_000 {
		name := filepath.Join(target, "file-"+strconv.Itoa(index)+".go")
		g.Expect(os.WriteFile(name, []byte("package fixture\n"), 0o600)).To(Succeed())
	}

	//nolint:gosec // The executable is this test binary, not user input.
	command := exec.Command(os.Args[0], "-test.run=^TestProgressSignalHelper$")

	command.Env = append(
		os.Environ(),
		"CODEVIZ_SIGNAL_HELPER=1",
		"CODEVIZ_SIGNAL_TARGET="+target,
		"CODEVIZ_SIGNAL_OUTPUT="+filepath.Join(target, "out.png"),
	)

	stderr, err := command.StderrPipe()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(command.Start()).To(Succeed())

	stageStarted := make(chan struct{})
	scanDone := make(chan struct{})

	var lines []string

	go func() {
		defer close(scanDone)

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			lines = append(lines, line)

			if strings.Contains(line, "Acquiring data: started") {
				select {
				case <-stageStarted:
				default:
					close(stageStarted)
				}
			}
		}
	}()

	select {
	case <-stageStarted:
	case <-time.After(10 * time.Second):
		_ = command.Process.Kill()

		t.Fatal("timed out waiting for acquisition progress")
	}

	g.Expect(command.Process.Signal(os.Interrupt)).To(Succeed())
	err = command.Wait()

	var exitErr *exec.ExitError

	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.As(err, &exitErr)).To(BeTrue())

	if exitErr == nil {
		panic("signal helper returned a non-exit error")
	}

	g.Expect(exitErr.ExitCode()).To(Equal(130))

	<-scanDone

	output := strings.Join(lines, "\n")
	g.Expect(output).To(ContainSubstring("Acquiring data: cancelled"))
	g.Expect(output).NotTo(ContainSubstring("Rendering"))
	g.Expect(output).NotTo(ContainSubstring("Writing output"))
}
