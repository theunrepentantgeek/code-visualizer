package scan_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
)

func TestIsBinaryFile_TextFile_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	path := filepath.Join(t.TempDir(), "hello.go")
	g.Expect(os.WriteFile(path, []byte("package main\n\nfunc main() {}\n"), 0o600)).To(Succeed())

	binary, err := scan.IsBinaryFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(binary).To(BeFalse())
}

func TestIsBinaryFile_BinaryFile_ReturnsTrue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	path := filepath.Join(t.TempDir(), "image.png")
	// PNG header contains null bytes
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00}
	g.Expect(os.WriteFile(path, data, 0o600)).To(Succeed())

	binary, err := scan.IsBinaryFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(binary).To(BeTrue())
}

func TestIsBinaryFile_EmptyFile_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	path := filepath.Join(t.TempDir(), "empty.txt")
	g.Expect(os.WriteFile(path, []byte{}, 0o600)).To(Succeed())

	binary, err := scan.IsBinaryFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(binary).To(BeFalse())
}

func TestIsBinaryFile_UTF16LE_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	path := filepath.Join(t.TempDir(), "utf16le.txt")
	// UTF-16 LE BOM followed by 'H' 'i' in UTF-16 LE (contains null bytes but is text)
	data := []byte{0xFF, 0xFE, 0x48, 0x00, 0x69, 0x00}
	g.Expect(os.WriteFile(path, data, 0o600)).To(Succeed())

	binary, err := scan.IsBinaryFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(binary).To(BeFalse())
}

func TestIsBinaryFile_UTF16BE_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	path := filepath.Join(t.TempDir(), "utf16be.txt")
	// UTF-16 BE BOM followed by 'H' 'i' in UTF-16 BE
	data := []byte{0xFE, 0xFF, 0x00, 0x48, 0x00, 0x69}
	g.Expect(os.WriteFile(path, data, 0o600)).To(Succeed())

	binary, err := scan.IsBinaryFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(binary).To(BeFalse())
}

func TestIsBinaryFile_NonexistentFile_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := scan.IsBinaryFile("/nonexistent/path/file.bin")
	g.Expect(err).To(HaveOccurred())
}
