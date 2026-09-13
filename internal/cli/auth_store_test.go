package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/steipete/spogo/internal/config"
	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/testutil"
)

func TestMain(m *testing.M) {
	if os.Getenv("SPOGO_TEST_TRASH_HELPER") == "1" {
		if len(os.Args) != 2 {
			os.Exit(2)
		}
		if err := os.Remove(os.Args[1]); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestAuthClearCmdNoPath(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	cmd := AuthClearCmd{}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestAuthClearCmdSuccess(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	dir := t.TempDir()
	ctx.Config = config.Default()
	ctx.ConfigPath = filepath.Join(dir, "config.toml")
	ctx.ProfileKey = "default"
	path := filepath.Join(dir, "cookies", "default.json")
	ctx.Profile.CookiePath = path
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("[]"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	name := "trash"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o755); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("SPOGO_TEST_TRASH_HELPER", "1")
	cmd := AuthClearCmd{}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if ctx.Profile.CookiePath != "" {
		t.Fatalf("expected profile cookie path cleared, got %q", ctx.Profile.CookiePath)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected helper to remove cookie file, got %v", err)
	}
}

func TestTrashFileMissing(t *testing.T) {
	t.Setenv("PATH", "")
	if err := trashFile("/tmp/missing"); err == nil {
		t.Fatalf("expected error")
	}
}
