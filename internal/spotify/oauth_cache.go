package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type OAuthTokenStatus struct {
	Exists     bool
	ClientID   string
	Scopes     []string
	ExpiresAt  time.Time
	Expired    bool
	HasRefresh bool
	FileMode   os.FileMode
}

func LoadOAuthToken(path string) (OAuthToken, error) {
	if _, err := os.Stat(path); err != nil {
		return OAuthToken{}, err
	}
	cacheLock, err := acquireOAuthCacheLock(context.Background(), path)
	if err != nil {
		return OAuthToken{}, err
	}
	defer releaseOAuthCacheLock(cacheLock)
	return loadOAuthTokenUnlocked(path)
}

func loadOAuthTokenUnlocked(path string) (OAuthToken, error) {
	info, err := os.Stat(path)
	if err != nil {
		return OAuthToken{}, err
	}
	if runtime.GOOS != "windows" {
		dirInfo, err := os.Stat(filepath.Dir(path))
		if err != nil {
			return OAuthToken{}, err
		}
		if dirInfo.Mode().Perm()&0o077 != 0 {
			return OAuthToken{}, fmt.Errorf("oauth token cache directory permissions are %04o; require 0700", dirInfo.Mode().Perm())
		}
	}
	if !info.Mode().IsRegular() {
		return OAuthToken{}, errors.New("oauth token cache is not a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return OAuthToken{}, fmt.Errorf("oauth token cache permissions are %04o; require 0600", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return OAuthToken{}, err
	}
	var token OAuthToken
	if err := json.Unmarshal(data, &token); err != nil {
		return OAuthToken{}, fmt.Errorf("decode oauth token cache: %w", err)
	}
	if token.RefreshToken == "" && token.AccessToken == "" {
		return OAuthToken{}, errors.New("oauth token cache contains no tokens")
	}
	return token, nil
}

func SaveOAuthToken(path string, token OAuthToken) error {
	cacheLock, err := acquireOAuthCacheLock(context.Background(), path)
	if err != nil {
		return err
	}
	defer releaseOAuthCacheLock(cacheLock)
	return saveOAuthTokenUnlocked(path, token)
}

func saveOAuthTokenUnlocked(path string, token OAuthToken) error {
	if path == "" {
		return errors.New("oauth token cache path is required")
	}
	if token.RefreshToken == "" {
		return errors.New("refusing to cache oauth token without refresh token")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dir, 0o700); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(dir, ".oauth-token-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		_ = tmp.Close()
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := replaceOAuthTokenFile(tmpPath, path); err != nil {
		return err
	}
	committed = true
	return nil
}

func OAuthStatus(path string) (OAuthTokenStatus, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return OAuthTokenStatus{}, nil
		}
		return OAuthTokenStatus{}, err
	}
	cacheLock, err := acquireOAuthCacheLock(context.Background(), path)
	if err != nil {
		return OAuthTokenStatus{}, err
	}
	defer releaseOAuthCacheLock(cacheLock)
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return OAuthTokenStatus{}, nil
		}
		return OAuthTokenStatus{}, err
	}
	token, err := loadOAuthTokenUnlocked(path)
	if err != nil {
		return OAuthTokenStatus{}, err
	}
	scopes := strings.Fields(token.Scope)
	return OAuthTokenStatus{
		Exists:     true,
		ClientID:   token.ClientID,
		Scopes:     scopes,
		ExpiresAt:  token.ExpiresAt,
		Expired:    !token.ExpiresAt.IsZero() && !token.ExpiresAt.After(time.Now()),
		HasRefresh: token.RefreshToken != "",
		FileMode:   info.Mode().Perm(),
	}, nil
}

func ClearOAuthToken(path string) error {
	if path == "" {
		return errors.New("oauth token cache path is required")
	}
	cacheLock, err := acquireOAuthCacheLock(context.Background(), path)
	if err != nil {
		return err
	}
	defer releaseOAuthCacheLock(cacheLock)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
