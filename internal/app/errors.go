package app

import (
	"errors"
	"fmt"
	"net"

	"github.com/alecthomas/kong"
	"github.com/steipete/spogo/internal/cookies"
	"github.com/steipete/spogo/internal/spotify"
)

type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit %d", e.Code)
	}
	return e.Err.Error()
}

func (e ExitError) Unwrap() error {
	return e.Err
}

func WrapExit(code int, err error) error {
	if err == nil {
		return nil
	}
	return ExitError{Code: code, Err: err}
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr ExitError
	if errors.As(err, &exitErr) {
		if exitErr.Code != 0 {
			return exitErr.Code
		}
	}
	var parseErr *kong.ParseError
	if errors.As(err, &parseErr) {
		return 2
	}
	if errors.Is(err, cookies.ErrNoCookies) {
		return 3
	}
	var apiErr spotify.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Status {
		case 401, 403:
			return 3
		default:
			return 1
		}
	}
	if isNetErr(err) {
		return 4
	}
	return 1
}

func isNetErr(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}
