//go:build linux || darwin

package scan

import (
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

	root, err := Scan(tmp)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).ToNot(BeNil())

	if root == nil {
		return
	}

	g.Expect(len(root.Files)).To(BeNumerically(">=", 1))
}
