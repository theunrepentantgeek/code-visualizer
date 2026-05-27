package main

import (
	"io"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func TestHelpMetricsCmdRun_GroupsMetricsByProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	output := captureStdout(t, func() {
		err := (HelpMetricsCmd{}).Run(&Flags{})
		g.Expect(err).NotTo(HaveOccurred())
	})

	filesystemIndex := strings.Index(output, "Filesystem metrics")
	gitIndex := strings.Index(output, "Git metrics")
	goIndex := strings.Index(output, "Go metrics")
	fileSizeIndex := strings.Index(output, "file-size")
	commitCountIndex := strings.Index(output, "commit-count")
	typeCountIndex := strings.Index(output, "type-count")

	g.Expect(filesystemIndex).To(BeNumerically(">=", 0))
	g.Expect(gitIndex).To(BeNumerically(">", filesystemIndex))
	g.Expect(goIndex).To(BeNumerically(">", gitIndex))
	g.Expect(fileSizeIndex).To(BeNumerically(">", filesystemIndex))
	g.Expect(fileSizeIndex).To(BeNumerically("<", gitIndex))
	g.Expect(commitCountIndex).To(BeNumerically(">", gitIndex))
	g.Expect(commitCountIndex).To(BeNumerically("<", goIndex))
	g.Expect(typeCountIndex).To(BeNumerically(">", goIndex))
	g.Expect(output).To(ContainSubstring("commit-count"))
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
