package spotify

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

func GenerateOAuthPKCE() (verifier, challenge string, err error) {
	verifier, err = randomURLSafe(64)
	if err != nil {
		return "", "", err
	}
	digest := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(digest[:]), nil
}

func GenerateOAuthState() (string, error) {
	return randomURLSafe(32)
}

func randomURLSafe(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func ValidateOAuthRedirectURI(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid spotify redirect URI: %w", err)
	}
	if parsed.Scheme != "http" {
		return errors.New("spotify CLI redirect URI must use http on a loopback IP")
	}
	host := parsed.Hostname()
	if host != "127.0.0.1" && host != "::1" {
		return errors.New("spotify CLI redirect URI must use 127.0.0.1 or [::1], not localhost or a non-loopback host")
	}
	port := parsed.Port()
	if port == "" {
		return errors.New("spotify CLI redirect URI must include a port")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return errors.New("spotify CLI redirect URI must use a port from 1 to 65535")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return errors.New("spotify CLI redirect URI cannot include userinfo, a query, or a fragment")
	}
	return nil
}
