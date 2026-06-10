package cli

import (
	"bufio"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steipete/spogo/internal/config"
	"github.com/steipete/spogo/internal/cookies"
	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/testutil"
)

func TestNormalizeCookieDomain(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ".spotify.com"},
		{"spotify.com", ".spotify.com"},
		{".spotify.com", ".spotify.com"},
		{"https://open.spotify.com/", ".open.spotify.com"},
	}
	for _, tc := range cases {
		if got := normalizeCookieDomain(tc.in); got != tc.want {
			t.Fatalf("normalizeCookieDomain(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeCookieValue(t *testing.T) {
	if got, ok := extractNamedCookieValue("sp_dc=token; Path=/; Secure", "sp_dc"); !ok || got != "token" {
		t.Fatalf("expected token, got %q", got)
	}
	if got := normalizePromptCookieValue("\"token\"", "sp_dc"); got != "token" {
		t.Fatalf("expected token, got %q", got)
	}
	if got := normalizePromptCookieValue("token", "sp_dc"); got != "token" {
		t.Fatalf("expected token, got %q", got)
	}
}

func TestReadPromptCookieValueEOF(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("sp_dc=token"))
	value, err := readPromptCookieValue(reader, nil, "sp_dc", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "token" {
		t.Fatalf("expected token, got %q", value)
	}
}

func TestReadPromptCookieValueRequiredRejectsEmpty(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	_, err := readPromptCookieValue(reader, nil, "sp_dc", true)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestPromptPastedCookies(t *testing.T) {
	ctx, _, errOut := testutil.NewTestContext(t, output.FormatPlain)
	withStdin(t, "sp_dc=token\nsp_key=key\nsp_t=device\n", func() {
		values, err := promptPastedCookies(ctx.Output)
		if err != nil {
			t.Fatalf("prompt: %v", err)
		}
		if values.spdc != "token" || values.spkey != "key" || values.spt != "device" {
			t.Fatalf("unexpected values: %#v", values)
		}
	})
	for _, prompt := range []string{"Paste sp_dc value:", "Paste sp_key value:", "Paste sp_t value:"} {
		if !strings.Contains(errOut.String(), prompt) {
			t.Fatalf("expected prompt %q in %q", prompt, errOut.String())
		}
	}
}

func TestPromptPastedCookiesAllowsEmptyOptionalValues(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	withStdin(t, "sp_dc=token\n\n\n", func() {
		values, err := promptPastedCookies(ctx.Output)
		if err != nil {
			t.Fatalf("prompt: %v", err)
		}
		if values.spdc != "token" || values.spkey != "" || values.spt != "" {
			t.Fatalf("unexpected values: %#v", values)
		}
	})
}

func TestParsePastedCookiesAnyOrder(t *testing.T) {
	values, err := parsePastedCookies(strings.NewReader("sp_t=device\nsp_dc=token\nsp_key=key\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if values.spdc != "token" || values.spkey != "key" || values.spt != "device" {
		t.Fatalf("unexpected values: %#v", values)
	}
}

func TestAuthPasteCmdFromStdin(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.Config = config.Default()
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"

	dest := filepath.Join(t.TempDir(), "out.json")
	withStdin(t, "sp_t=device\nsp_dc=token\nsp_key=key\n", func() {
		cmd := AuthPasteCmd{CookiePath: dest, Domain: "spotify.com"}
		if err := cmd.Run(ctx); err != nil {
			t.Fatalf("run: %v", err)
		}
	})
	cookiesList, err := cookies.Read(dest)
	if err != nil {
		t.Fatalf("read cookies: %v", err)
	}
	if len(cookiesList) != 3 {
		t.Fatalf("expected 3 cookies, got %d", len(cookiesList))
	}
	if ctx.Profile.CookiePath != dest {
		t.Fatalf("expected profile cookie path %s, got %s", dest, ctx.Profile.CookiePath)
	}
}

func TestAuthPasteCmdWarnsWhenMissingSPT(t *testing.T) {
	ctx, _, errOut := testutil.NewTestContext(t, output.FormatPlain)
	ctx.Config = config.Default()
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	ctx.Profile = config.Profile{Engine: "connect"}

	dest := filepath.Join(t.TempDir(), "out.json")
	withStdin(t, "sp_dc=token\n", func() {
		cmd := AuthPasteCmd{CookiePath: dest}
		if err := cmd.Run(ctx); err != nil {
			t.Fatalf("run: %v", err)
		}
	})
	if !strings.Contains(errOut.String(), "missing sp_t") {
		t.Fatalf("expected warning in stderr, got %q", errOut.String())
	}
}

func TestAuthPasteCmdRequiresSPDC(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	withStdin(t, "", func() {
		cmd := AuthPasteCmd{}
		if err := cmd.Run(ctx); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestAuthPasteCmdNoInputFromStdin(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.Config = config.Default()
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	ctx.Settings.NoInput = true

	dest := filepath.Join(t.TempDir(), "out.json")
	withStdin(t, "sp_dc=token\nsp_t=device\n", func() {
		cmd := AuthPasteCmd{CookiePath: dest}
		if err := cmd.Run(ctx); err != nil {
			t.Fatalf("run: %v", err)
		}
	})
	if _, err := cookies.Read(dest); err != nil {
		t.Fatalf("expected cookies file")
	}
}
