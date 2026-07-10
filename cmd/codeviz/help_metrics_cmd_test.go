package main

import (
	"io"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	. "github.com/onsi/gomega"
)

//nolint:paralleltest // captureStdout swaps global os.Stdout, so this test cannot run in parallel
func TestHelpMetricsCmdRun_GroupsMetricsByProvider(t *testing.T) {
	g := NewGomegaWithT(t)

	output := captureStdout(t, func() {
		err := (HelpMetricsCmd{}).Run(&Flags{})
		g.Expect(err).NotTo(HaveOccurred())
	})

	syntaxIndex := strings.Index(output, "Syntax:")
	filesystemIndex := strings.Index(output, "Filesystem metrics")
	gitIndex := strings.Index(output, "Git metrics")
	goIndex := strings.Index(output, "Go metrics")

	g.Expect(syntaxIndex).To(BeNumerically(">=", 0))
	g.Expect(filesystemIndex).To(BeNumerically(">=", 0))
	g.Expect(gitIndex).To(BeNumerically(">", filesystemIndex))
	g.Expect(goIndex).To(BeNumerically(">", gitIndex))

	filesystemSection := output[filesystemIndex:gitIndex]
	goSection := output[goIndex:]

	g.Expect(filesystemSection).To(ContainSubstring("file-size"))
	g.Expect(filesystemSection).To(ContainSubstring("sum"))
	g.Expect(filesystemSection).To(ContainSubstring("min"))
	g.Expect(goSection).To(MatchRegexp(`public \(Exported\s+declarations only\)`))
	g.Expect(goSection).To(MatchRegexp(`Filters:\s+public,\s+private`))
	// Legacy "Other metrics" section no longer exists
	g.Expect(output).NotTo(ContainSubstring("Other metrics"))
}

func TestHelpMetricsCmdRun_WrapsOutputToConsoleWidth(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Setenv("COLUMNS", "30")

	output := captureStdout(t, func() {
		err := (HelpMetricsCmd{}).Run(&Flags{})
		g.Expect(err).NotTo(HaveOccurred())
	})

	for line := range strings.SplitSeq(output, "\n") {
		g.Expect(utf8.RuneCountInString(line)).To(BeNumerically("<=", 30), "line exceeds console width: %q", line)
	}
}

func captureStdout(t *testing.T, run func()) string {
	t.Helper()

	oldStdout := os.Stdout

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}

	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
		os.Stdout = oldStdout
	})

	os.Stdout = writer

	dataCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		data, readErr := io.ReadAll(reader)
		if readErr != nil {
			errCh <- readErr

			return
		}

		dataCh <- data
	}()

	run()

	_ = writer.Close()
	os.Stdout = oldStdout

	select {
	case readErr := <-errCh:
		t.Fatalf("read stdout: %v", readErr)
	case data := <-dataCh:
		return string(data)
	}

	return ""
}
