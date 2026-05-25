package main

import (
	"io"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func TestHelpMetricsCmdRun_GroupsMetricsByProvider(t *testing.T) {
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
	g.Expect(output).To(ContainSubstring("† requires a git repository"))
}

func captureStdout(t *testing.T, run func()) string {
	t.Helper()

	oldStdout := os.Stdout

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}

	os.Stdout = writer
	defer func() {
		os.Stdout = oldStdout
	}()

	run()

	_ = writer.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}

	return string(data)
}
