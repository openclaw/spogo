package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steipete/spogo/internal/cookies"
)

func TestRunHelp(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := run([]string{"--help"}, out, errOut)
	if code != 0 {
		t.Fatalf("expected 0, got %d; out=%q err=%q", code, out.String(), errOut.String())
	}
	if out.Len() == 0 {
		t.Fatalf("expected help output")
	}
}

func TestRunBadArgs(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := run([]string{"nope"}, out, errOut)
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestRunVersion(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := run([]string{"--version"}, out, errOut)
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunInvalidConfig(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	path := filepath.Join(t.TempDir(), "bad.toml")
	if err := os.WriteFile(path, []byte("not=toml=\""), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	code := run([]string{"--config", path, "auth", "status"}, out, errOut)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunInvalidProfile(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := run([]string{"--market", "USA", "queue", "clear"}, out, errOut)
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestRunCompletionFish(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := run([]string{"completion", "fish"}, out, errOut)
	if code != 0 {
		t.Fatalf("expected 0, got %d; out=%q err=%q", code, out.String(), errOut.String())
	}
	got := out.String()
	for _, want := range []string{
		"# fish completion for spogo",
		"complete -c spogo",
		"function __spogo_seen_command_path",
		"-a 'completion'",
		"-l config",
		"-a 'fish'",
		"-n '__spogo_seen_command_path \\'playlist\\' \\'tracks\\'' -l limit",
		"-n '__spogo_seen_command_path \\'library\\' \\'tracks\\' \\'list\\'' -l limit",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("completion output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "__fish_seen_subcommand_from \\'playlist\\'") {
		t.Fatalf("completion output uses leaf-only playlist condition:\n%s", got)
	}
}

func TestRunCompletionBypassesProfileAndConfigValidation(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	path := filepath.Join(t.TempDir(), "bad.toml")
	if err := os.WriteFile(path, []byte("not=toml=\""), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	code := run([]string{"--config", path, "--market", "USA", "completion", "fish"}, out, errOut)
	if code != 0 {
		t.Fatalf("expected 0, got %d; out=%q err=%q", code, out.String(), errOut.String())
	}
}

func TestRunCompletionUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"completion"},
		{"completion", "bash"},
	} {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := run(args, out, errOut)
		if code != 2 {
			t.Fatalf("run(%v): expected 2, got %d; out=%q err=%q", args, code, out.String(), errOut.String())
		}
	}
}

func TestRunCommandError(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := run([]string{"queue", "clear"}, out, errOut)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunAuthStatus(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	dir := t.TempDir()
	cookiePath := filepath.Join(dir, "cookies.json")
	if err := cookies.Write(cookiePath, []*http.Cookie{{Name: "sp_dc", Value: "token"}}); err != nil {
		t.Fatalf("cookies: %v", err)
	}
	configPath := filepath.Join(dir, "config.toml")
	config := []byte(fmt.Sprintf("default_profile = \"default\"\n[profile.default]\ncookie_path = %q\n", cookiePath))
	if err := os.WriteFile(configPath, config, 0o644); err != nil {
		t.Fatalf("config: %v", err)
	}
	code := run([]string{"--config", configPath, "auth", "status"}, out, errOut)
	if code != 0 {
		t.Fatalf("expected 0, got %d; out=%q err=%q", code, out.String(), errOut.String())
	}
}

func TestNormalizeArgsMovesNoInput(t *testing.T) {
	got := normalizeArgs([]string{"auth", "paste", "--no-input", "--cookie-path", "cookies.json"})
	want := []string{"--no-input", "auth", "paste", "--cookie-path", "cookies.json"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMain(t *testing.T) {
	origArgs := os.Args
	origExit := exitFunc
	defer func() {
		os.Args = origArgs
		exitFunc = origExit
	}()
	os.Args = []string{"spogo", "--help"}
	got := -1
	exitFunc = func(code int) { got = code }
	main()
	if got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}
