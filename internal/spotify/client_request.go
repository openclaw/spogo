package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

func (c *Client) get(ctx context.Context, path string, params url.Values, dest any) error {
	return c.send(ctx, http.MethodGet, path, params, nil, dest)
}

func (c *Client) put(ctx context.Context, path string, payload any) error {
	return c.send(ctx, http.MethodPut, path, nil, payload, nil)
}

func (c *Client) postJSON(ctx context.Context, path string, payload any, dest any) error {
	return c.send(ctx, http.MethodPost, path, nil, payload, dest)
}

func (c *Client) send(ctx context.Context, method, path string, params url.Values, payload any, dest any) error {
	const (
		maxAttempts   = 3
		maxRetryDelay = 3 * time.Second
	)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		requestURL := c.baseURL + path
		if params == nil {
			if c.market != "" || c.language != "" {
				params = url.Values{}
			}
		}
		if params != nil {
			if c.market != "" && params.Get("market") == "" {
				params.Set("market", c.market)
			}
			if c.language != "" && params.Get("locale") == "" {
				params.Set("locale", c.language)
			}
			if encoded := params.Encode(); encoded != "" {
				requestURL += "?" + encoded
			}
		}
		var body io.Reader
		if payload != nil {
			data, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			body = bytes.NewReader(data)
		}
		req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
		if err != nil {
			return err
		}
		token, err := c.token(ctx)
		if err != nil {
			return err
		}
		contentType := ""
		if payload != nil {
			contentType = "application/json"
		}
		applyRequestHeaders(req, requestHeaders{
			AccessToken: token,
			Accept:      "application/json",
			ContentType: contentType,
		})
		resp, err := c.client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxAttempts-1 {
			retryAfter := retryAfterFromResponse(resp)
			if retryAfter <= 0 {
				retryAfter = time.Second
			}
			if retryAfter <= maxRetryDelay {
				_ = resp.Body.Close()
				select {
				case <-time.After(retryAfter):
					continue
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}

		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == http.StatusNoContent {
			if dest != nil {
				return ErrNoContent
			}
			return nil
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return apiErrorFromResponse(resp)
		}
		if dest == nil {
			return nil
		}
		if resp.ContentLength == 0 {
			return nil
		}
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return errors.New("spotify api error (429): rate limit retry exhausted")
}
