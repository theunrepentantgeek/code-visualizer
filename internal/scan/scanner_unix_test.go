//go:build linux || darwin

package scan

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestScanPermissionDenied(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "readable.txt"))
	g.Expect(err).NotTo(HaveOccurred())
	f.WriteString("hello") //nolint:errcheck // test data
	f.Close()

	unreadable := filepath.Join(tmp, "unreadable.txt")
	err = os.WriteFile(unreadable, []byte("secret"), 0o000)
	g.Expect(err).NotTo(HaveOccurred())

	root, err := Scan(context.Background(), tmp, nil, nil, true)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).ToNot(BeNil())

	if root == nil {
		return
	}

	g.Expect(len(root.Files)).To(BeNumerically(">=", 1))
}

func TestScanSkipsFileSymlinkOutsideRoot(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	parent := t.TempDir()
	rootPath := filepath.Join(parent, "root")
	g.Expect(os.Mkdir(rootPath, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(rootPath, "safe.txt"), []byte("safe\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(parent, "outside.txt"), []byte("outside\n"), 0o600)).To(Succeed())
	g.Expect(os.Symlink("../outside.txt", filepath.Join(rootPath, "outside-link.txt"))).To(Succeed())

	root, err := Scan(context.Background(), rootPath, nil, nil, true)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).NotTo(BeNil())

	if root == nil {
		t.Fatal("expected scanned root")
	}

	g.Expect(collectFileNames(root)).To(ConsistOf("safe.txt"))
}
