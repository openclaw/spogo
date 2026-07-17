package spotify

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAPIErrorFromResponse(t *testing.T) {
	body := io.NopCloser(strings.NewReader(`{"error":{"status":401,"message":"bad"}}`))
	resp := &http.Response{StatusCode: 401, Status: "401", Body: body}
	err := apiErrorFromResponse(resp)
	var apiErr APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError")
	}
	if apiErr.Status != 401 || apiErr.Message != "bad" {
		t.Fatalf("unexpected: %#v", apiErr)
	}
}

func TestAPIErrorFromResponseNil(t *testing.T) {
	err := apiErrorFromResponse(nil)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestAPIErrorFromResponseRetryAfter(t *testing.T) {
	body := io.NopCloser(strings.NewReader(`{"error":{"status":429,"message":"rate limit"}}`))
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Status:     "429",
		Header:     http.Header{"Retry-After": []string{"42"}},
		Body:       body,
	}
	err := apiErrorFromResponse(resp)
	var apiErr APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError")
	}
	if apiErr.RetryAfter != 42*time.Second {
		t.Fatalf("retry after = %v, want 42s", apiErr.RetryAfter)
	}
	if !strings.Contains(apiErr.Error(), "retry-after hint 42s") {
		t.Fatalf("expected retry-after hint in error string, got %q", apiErr.Error())
	}
}

func TestAPIErrorFromResponseRetryAfterHTTPDate(t *testing.T) {
	when := time.Now().Add(30 * time.Second).UTC().Format(http.TimeFormat)
	body := io.NopCloser(strings.NewReader(`{"error":{"status":429,"message":"rate limit"}}`))
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Status:     "429",
		Header:     http.Header{"Retry-After": []string{when}},
		Body:       body,
	}
	err := apiErrorFromResponse(resp)
	var apiErr APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError")
	}
	// HTTP-date has one-second granularity, so allow a small window below 30s.
	if apiErr.RetryAfter <= 25*time.Second || apiErr.RetryAfter > 30*time.Second {
		t.Fatalf("retry after = %v, want ~30s from HTTP-date", apiErr.RetryAfter)
	}
}

func TestRetryAfterFromResponseRejectsDurationOverflow(t *testing.T) {
	resp := &http.Response{Header: http.Header{"Retry-After": []string{"9223372037"}}}
	if got := retryAfterFromResponse(resp); got != 0 {
		t.Fatalf("retry after = %v, want 0 for duration overflow", got)
	}
}

func TestAPIErrorError(t *testing.T) {
	err := APIError{Status: 400, Message: "bad"}
	if err.Error() == "" {
		t.Fatalf("expected error string")
	}
	err = APIError{Status: 400}
	if err.Error() == "" {
		t.Fatalf("expected error string")
	}
	err = APIError{}
	if err.Error() == "" {
		t.Fatalf("expected error string")
	}
}
