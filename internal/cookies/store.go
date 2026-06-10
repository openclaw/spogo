package cookies

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type StoredCookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain"`
	Path     string    `json:"path"`
	Expires  time.Time `json:"expires"`
	Secure   bool      `json:"secure"`
	HTTPOnly bool      `json:"http_only"`
}

func Read(path string) ([]*http.Cookie, error) {
	if path == "" {
		return nil, errors.New("cookie path required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var stored []StoredCookie
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}
	cookies := make([]*http.Cookie, 0, len(stored))
	for _, c := range stored {
		cookies = append(cookies, &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  c.Expires,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		})
	}
	return cookies, nil
}

func Write(path string, cookies []*http.Cookie) error {
	if path == "" {
		return errors.New("cookie path required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	stored := make([]StoredCookie, 0, len(cookies))
	for _, c := range cookies {
		if c == nil {
			continue
		}
		stored = append(stored, StoredCookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  jsonSafeExpiry(c.Expires),
			Secure:   c.Secure,
			HTTPOnly: c.HttpOnly,
		})
	}
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func jsonSafeExpiry(expires time.Time) time.Time {
	if expires.IsZero() {
		return time.Time{}
	}
	if _, err := expires.MarshalJSON(); err != nil {
		return time.Time{}
	}
	return expires
}
