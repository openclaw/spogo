package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steipete/spogo/internal/cookies"
	"github.com/steipete/sweetcookie"
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

func TestRunCompletionScripts(t *testing.T) {
	tests := []struct {
		shell string
		want  string
	}{
		{shell: "bash", want: "complete -o default -o bashdefault"},
		{shell: "zsh", want: "#compdef spogo"},
		{shell: "fish", want: "complete -f -c spogo"},
	}
	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			code := run([]string{"completion", tt.shell}, out, errOut)
			if code != 0 {
				t.Fatalf("expected 0, got %d; out=%q err=%q", code, out.String(), errOut.String())
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Fatalf("completion output missing %q:\n%s", tt.want, out.String())
			}
		})
	}
}

func TestRunCompletionBypassesProfileAndConfigValidation(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	path := filepath.Join(t.TempDir(), "bad.toml")
	if err := os.WriteFile(path, []byte("not=toml=\""), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	for _, shell := range []string{"bash", "zsh", "fish"} {
		out.Reset()
		errOut.Reset()
		code := run([]string{"--config", path, "--market", "USA", "completion", shell}, out, errOut)
		if code != 0 {
			t.Fatalf("%s: expected 0, got %d; out=%q err=%q", shell, code, out.String(), errOut.String())
		}
	}
}

func TestRunCompletionRequestBypassesNormalExecution(t *testing.T) {
	t.Setenv("COMP_LINE", "spogo playlist ")
	t.Setenv("COMP_POINT", "15")

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := run([]string{"--market", "USA", "queue", "clear"}, out, errOut)
	if code != 0 {
		t.Fatalf("expected 0, got %d; out=%q err=%q", code, out.String(), errOut.String())
	}
	for _, want := range []string{"tracks\n", "create\n", "remove\n"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("completion predictions missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunCompletionUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"completion"},
		{"completion", "powershell"},
	} {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := run(args, out, errOut)
		if code != 2 {
			t.Fatalf("run(%v): expected 2, got %d; out=%q err=%q", args, code, out.String(), errOut.String())
		}
	}
}

func TestRunCompletionHelpListsSupportedShells(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := run([]string{"completion", "--help"}, out, errOut)
	if code != 0 {
		t.Fatalf("expected 0, got %d; out=%q err=%q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "bash, zsh, or fish") {
		t.Fatalf("completion help does not list supported shells:\n%s", out.String())
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

func TestRunAuthStatusWithoutCookiesReturnsAuthExitCode(t *testing.T) {
	restore := cookies.SetReadCookies(func(context.Context, sweetcookie.Options) (sweetcookie.Result, error) {
		return sweetcookie.Result{}, nil
	})
	defer restore()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	configPath := filepath.Join(t.TempDir(), "config.toml")
	code := run([]string{"--config", configPath, "--no-input", "auth", "status"}, out, errOut)
	if code != 3 {
		t.Fatalf("expected 3, got %d; out=%q err=%q", code, out.String(), errOut.String())
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
