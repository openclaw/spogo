package cookies

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteReadCookies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.json")
	expires := time.Now().Add(time.Hour).UTC()
	input := []*http.Cookie{{
		Name:     "sp_dc",
		Value:    "token",
		Domain:   ".spotify.com",
		Path:     "/",
		Expires:  expires,
		Secure:   true,
		HttpOnly: true,
	}}
	if err := Write(path, input); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat: %v", err)
	}
	out, err := Read(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(out) != 1 || out[0].Name != "sp_dc" {
		t.Fatalf("unexpected cookies: %#v", out)
	}
	if !out[0].Expires.Equal(expires) {
		t.Fatalf("expected expiry %v, got %v", expires, out[0].Expires)
	}
}

func TestWriteErrors(t *testing.T) {
	if err := Write("", nil); err == nil {
		t.Fatalf("expected error")
	}
}

func TestReadErrors(t *testing.T) {
	if _, err := Read(""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWriteSkipsNilCookie(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.json")
	input := []*http.Cookie{
		nil,
		{Name: "sp_dc", Value: "token"},
	}
	if err := Write(path, input); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, err := Read(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(out) != 1 || out[0].Name != "sp_dc" {
		t.Fatalf("unexpected cookies: %#v", out)
	}
}

func TestWriteDropsUnmarshalableExpiry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.json")
	validExpires := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	input := []*http.Cookie{
		{
			Name:    "sp_dc",
			Value:   "token",
			Expires: time.Unix(1<<62, 0).UTC(),
		},
		{
			Name:    "sp_t",
			Value:   "token",
			Expires: validExpires,
		},
	}
	if err := Write(path, input); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, err := Read(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(out) != 2 || out[0].Name != "sp_dc" || out[1].Name != "sp_t" {
		t.Fatalf("unexpected cookies: %#v", out)
	}
	if !out[0].Expires.IsZero() {
		t.Fatalf("expected zero expiry, got %v", out[0].Expires)
	}
	if !out[1].Expires.Equal(validExpires) {
		t.Fatalf("expected valid expiry %v, got %v", validExpires, out[1].Expires)
	}
}

func TestReadBadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Read(path); err == nil {
		t.Fatalf("expected error")
	}
}
