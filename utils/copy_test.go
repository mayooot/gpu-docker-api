package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDirKeepsHiddenFiles(t *testing.T) {
	src, dest := t.TempDir(), t.TempDir()

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(src, "root", ".cache"), 0755))
	must(os.WriteFile(filepath.Join(src, ".bashrc"), []byte("alias ll='ls -l'"), 0644))
	must(os.WriteFile(filepath.Join(src, "root", ".cache", "pip"), []byte("x"), 0600))
	must(os.Symlink(".bashrc", filepath.Join(src, "link")))

	must(CopyDir(src, dest))

	for _, p := range []string{".bashrc", "root/.cache/pip"} {
		if _, err := os.Stat(filepath.Join(dest, p)); err != nil {
			t.Errorf("%s was not copied: %v", p, err)
		}
	}
	if target, err := os.Readlink(filepath.Join(dest, "link")); err != nil || target != ".bashrc" {
		t.Errorf("symlink not preserved, target %q, err %v", target, err)
	}
	if fi, err := os.Stat(filepath.Join(dest, "root", ".cache", "pip")); err == nil && fi.Mode().Perm() != 0600 {
		t.Errorf("file mode not preserved, got %v", fi.Mode().Perm())
	}
}
