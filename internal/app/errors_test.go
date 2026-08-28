package app

import (
	"errors"
	"net"
	"testing"

	"github.com/steipete/spogo/internal/cookies"
	"github.com/steipete/spogo/internal/spotify"
)

func TestExitError(t *testing.T) {
	err := ExitError{Code: 7, Err: errors.New("boom")}
	if err.Error() != "boom" {
		t.Fatalf("error: %s", err.Error())
	}
	if err.Unwrap() == nil {
		t.Fatalf("expected unwrap")
	}
}

func TestExitErrorNilErr(t *testing.T) {
	err := ExitError{Code: 9}
	if err.Error() == "" {
		t.Fatalf("expected message")
	}
}

func TestWrapExit(t *testing.T) {
	if err := WrapExit(1, nil); err != nil {
		t.Fatalf("expected nil")
	}
	wrapped := WrapExit(2, errors.New("boom"))
	if ExitCode(wrapped) != 2 {
		t.Fatalf("expected 2")
	}
}

func TestExitCode(t *testing.T) {
	if ExitCode(nil) != 0 {
		t.Fatalf("expected 0")
	}
	if ExitCode(ExitError{Code: 5}) != 5 {
		t.Fatalf("expected 5")
	}
	if ExitCode(spotify.APIError{Status: 401}) != 3 {
		t.Fatalf("expected 3")
	}
	if ExitCode(cookies.ErrNoCookies) != 3 {
		t.Fatalf("expected 3")
	}
	if ExitCode(spotify.ErrOAuthAuthentication) != 3 {
		t.Fatalf("expected 3")
	}
	if ExitCode(spotify.APIError{Status: 500}) != 1 {
		t.Fatalf("expected 1")
	}
	if ExitCode(&net.DNSError{IsTimeout: true}) != 4 {
		t.Fatalf("expected 4")
	}
	if ExitCode(errors.New("oops")) != 1 {
		t.Fatalf("expected 1")
	}
}
