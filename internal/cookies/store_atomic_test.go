package cookies

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteReplacesPermissiveCookieFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file permissions")
	}
	path := filepath.Join(t.TempDir(), "cookies.json")
	if err := os.WriteFile(path, []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, []*http.Cookie{{Name: "sp_dc", Value: "synthetic-cookie"}}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("cookie permissions = %04o, want 0600", got)
	}
}

func TestWriteKeepsExistingReadersSnapshot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows readers block replacement until closed")
	}
	path := filepath.Join(t.TempDir(), "cookies.json")
	if err := Write(path, []*http.Cookie{{Name: "sp_dc", Value: "old-synthetic-cookie"}}); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	if err := Write(path, []*http.Cookie{{Name: "sp_dc", Value: "new-synthetic-cookie"}}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(snapshot) != string(original) {
		t.Fatal("cookie replacement changed the file held by an existing reader")
	}
	current, err := Read(path)
	if err != nil || len(current) != 1 || current[0].Value != "new-synthetic-cookie" {
		t.Fatalf("new cookie file unavailable: %v", err)
	}
}

func TestWritePreservesCookiePathSymlink(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new target", true: "existing target"}[existing], func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "target.json")
			if existing {
				if err := Write(target, nil); err != nil {
					t.Fatal(err)
				}
			}
			link := filepath.Join(dir, "cookies.json")
			if err := os.Symlink("target.json", link); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			if err := Write(link, []*http.Cookie{{Name: "sp_dc", Value: "synthetic-cookie"}}); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Readlink(link); err != nil {
				t.Fatalf("symlink replaced: %v", err)
			}
			got, err := Read(target)
			if err != nil || len(got) != 1 || got[0].Value != "synthetic-cookie" {
				t.Fatalf("target not updated: %v", err)
			}
		})
	}
}
