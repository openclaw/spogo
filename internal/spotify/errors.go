package spotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

var (
	ErrNoContent   = fmt.Errorf("no content")
	ErrUnsupported = errors.New("unsupported operation")
)

type APIError struct {
	Status  int
	Message string
	Body    string
	// RetryAfter is the cooldown Spotify advertised via the Retry-After header.
	// It is guidance for the earliest sensible retry, not a promise that the
	// next request will succeed: Spotify may answer a post-cooldown retry with
	// another 429 and a fresh Retry-After.
	RetryAfter time.Duration
}

func (e APIError) Error() string {
	suffix := ""
	if e.RetryAfter > 0 {
		suffix = fmt.Sprintf(" (retry-after hint %s)", e.RetryAfter.Round(time.Second))
	}
	if e.Message != "" {
		return fmt.Sprintf("spotify api error (%d): %s%s", e.Status, e.Message, suffix)
	}
	if e.Status != 0 {
		return fmt.Sprintf("spotify api error (%d)%s", e.Status, suffix)
	}
	return "spotify api error"
}

func apiErrorFromResponse(resp *http.Response) error {
	if resp == nil {
		return APIError{Message: "nil response"}
	}
	body, _ := io.ReadAll(resp.Body)
	payload := struct {
		Error struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}{
		Message: resp.Status,
	}
	_ = json.Unmarshal(body, &payload)
	status := resp.StatusCode
	message := payload.Error.Message
	if message == "" {
		message = payload.Message
	}
	return APIError{Status: status, Message: message, Body: string(body), RetryAfter: retryAfterFromResponse(resp)}
}

func retryAfterFromResponse(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	header := resp.Header.Get("Retry-After")
	if header == "" {
		return 0
	}
	if seconds, err := strconv.ParseInt(header, 10, 64); err == nil {
		const maxDurationSeconds = int64(^uint64(0)>>1) / int64(time.Second)
		if seconds <= 0 || seconds > maxDurationSeconds {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	when, err := http.ParseTime(header)
	if err != nil {
		return 0
	}
	delay := time.Until(when)
	if delay <= 0 {
		return 0
	}
	return delay
}

func preserveRateLimitHint(previous, current error) error {
	if current == nil {
		return nil
	}
	var currentAPIError APIError
	if errors.As(current, &currentAPIError) && currentAPIError.Status == http.StatusTooManyRequests && currentAPIError.RetryAfter > 0 {
		return current
	}
	var previousAPIError APIError
	if errors.As(previous, &previousAPIError) && previousAPIError.Status == http.StatusTooManyRequests && previousAPIError.RetryAfter > 0 {
		return previous
	}
	return current
}
