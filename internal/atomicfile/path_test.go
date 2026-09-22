package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathAliases(t *testing.T) {
	dir := t.TempDir()
	canonicalDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "private.json")
	for _, existing := range []bool{false, true} {
		if existing {
			if err := Write(path, nil); err != nil {
				t.Fatal(err)
			}
		}
		got, err := ResolvePath(path)
		if want := filepath.Join(canonicalDir, "private.json"); err != nil || got != want {
			t.Fatalf("resolved path = %q, %v; want %q", got, err, want)
		}
	}
	for _, target := range []string{"private.json", filepath.Join(dir, "new.json")} {
		link := filepath.Join(t.TempDir(), "alias")
		if !filepath.IsAbs(target) {
			link = filepath.Join(dir, "alias")
		}
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		got, err := ResolvePath(link)
		if want := filepath.Join(canonicalDir, filepath.Base(target)); err != nil || got != want {
			t.Fatalf("resolved alias = %q, %v; want %q", got, err, want)
		}
	}
}

func TestResolvePathErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := ResolvePath(filepath.Join(dir, "missing", "private.json")); err == nil {
		t.Fatal("expected missing parent error")
	}
	link := filepath.Join(dir, "cycle")
	if err := os.Symlink("cycle", link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := ResolvePath(link); err == nil {
		t.Fatal("expected cyclic symlink error")
	}
}
