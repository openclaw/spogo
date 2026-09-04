package app

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/steipete/spogo/internal/cookies"
	"github.com/steipete/spogo/internal/spotify"
)

type engineName string

const (
	engineConnect     engineName = "connect"
	engineWeb         engineName = "web"
	engineAuto        engineName = "auto"
	engineAppleScript engineName = "applescript"

	authCookies = "cookies"
	authOAuth   = "oauth"
)

func (c *Context) Spotify() (spotify.API, error) {
	if c == nil {
		return nil, errors.New("nil context")
	}
	if c.spotifyClient != nil {
		return c.spotifyClient, nil
	}
	source, err := c.cookieSource()
	if err != nil {
		return nil, err
	}
	client, err := c.buildSpotifyClient(source)
	if err != nil {
		return nil, err
	}
	c.spotifyClient = client
	return client, nil
}

func (c *Context) buildSpotifyClient(source cookies.Source) (spotify.API, error) {
	switch c.engine() {
	case engineConnect:
		webClient, err := c.newWebClient(source)
		if err != nil {
			return nil, err
		}
		return c.newConnectClient(source, webClient)
	case engineWeb:
		return c.newPlaybackFallbackClient(source)
	case engineAuto:
		return c.newAutoClient(source)
	case engineAppleScript:
		return c.newAppleScriptClient(source)
	default:
		return nil, fmt.Errorf("unknown engine %q (use auto, web, connect, or applescript)", c.engine())
	}
}

func (c *Context) newPlaybackFallbackClient(source cookies.Source) (spotify.API, error) {
	webClient, err := c.newWebClient(source)
	if err != nil {
		return nil, err
	}
	client := spotify.API(webClient)
	if connectClient, connectErr := c.newConnectClient(source, webClient); connectErr == nil {
		client = spotify.NewPlaybackFallbackClient(webClient, connectClient)
	}
	return client, nil
}

func (c *Context) newAutoClient(source cookies.Source) (spotify.API, error) {
	webClient, err := c.newWebClient(source)
	if err != nil {
		return nil, err
	}
	client := spotify.API(webClient)
	if connectClient, connectErr := c.newConnectClient(source, webClient); connectErr == nil {
		if localClient, localErr := spotify.NewAppleScriptClient(spotify.AppleScriptOptions{}); localErr == nil {
			client = spotify.NewAutoClient(connectClient, webClient, localClient)
		} else {
			client = spotify.NewAutoClient(connectClient, webClient)
		}
	}
	return client, nil
}

func (c *Context) newAppleScriptClient(source cookies.Source) (spotify.API, error) {
	var fallback spotify.API
	if webClient, webErr := c.newWebClient(source); webErr == nil {
		fallback = webClient
		if connectClient, connectErr := c.newConnectClient(source, webClient); connectErr == nil {
			fallback = spotify.NewPlaybackFallbackClient(webClient, connectClient)
		}
	}
	return spotify.NewAppleScriptClient(spotify.AppleScriptOptions{Fallback: fallback})
}

func (c *Context) newConnectClient(source cookies.Source, webClient *spotify.Client) (*spotify.ConnectClient, error) {
	return spotify.NewConnectClient(spotify.ConnectOptions{
		Source:    source,
		Market:    c.Profile.Market,
		Language:  c.Profile.Language,
		Device:    c.Profile.Device,
		Timeout:   c.Settings.Timeout,
		CachePath: c.ResolveCachePath(),
		WebClient: webClient,
	})
}

func (c *Context) newWebClient(source cookies.Source) (*spotify.Client, error) {
	var provider spotify.TokenProvider
	switch c.auth() {
	case authCookies:
		provider = spotify.CookieTokenProvider{Source: source, Timeout: c.Settings.Timeout}
	case authOAuth:
		oauthProvider, err := spotify.NewOAuthTokenProvider(spotify.OAuthOptions{
			ClientID:    c.Profile.SpotifyClientID,
			RedirectURI: c.Profile.SpotifyRedirectURI,
			CachePath:   c.ResolveOAuthTokenPath(),
			HTTPClient:  &http.Client{Timeout: c.EnsureTimeout()},
		})
		if err != nil {
			return nil, err
		}
		provider = oauthProvider
	default:
		return nil, fmt.Errorf("unknown auth %q (use cookies or oauth)", c.auth())
	}
	return spotify.NewClient(spotify.Options{
		TokenProvider: provider,
		Market:        c.Profile.Market,
		Language:      c.Profile.Language,
		Device:        c.Profile.Device,
		Timeout:       c.Settings.Timeout,
	})
}

func (c *Context) engine() engineName {
	engine := engineName(strings.ToLower(strings.TrimSpace(c.Profile.Engine)))
	if engine == "" {
		return engineConnect
	}
	return engine
}

func (c *Context) auth() string {
	auth := strings.ToLower(strings.TrimSpace(c.Profile.Auth))
	if auth == "" {
		return authCookies
	}
	return auth
}

func (c *Context) cookieSource() (cookies.Source, error) {
	if c.Profile.CookiePath != "" {
		return cookies.FileSource{Path: c.Profile.CookiePath}, nil
	}
	defaultPath := c.ResolveCookiePath()
	if defaultPath != "" {
		if _, err := os.Stat(defaultPath); err == nil {
			return cookies.FileSource{Path: defaultPath}, nil
		}
	}
	return cookies.BrowserSource{
		Browser: defaultBrowser(c.Profile.Browser),
		Profile: c.Profile.BrowserProfile,
		Domain:  "spotify.com",
	}, nil
}

func defaultBrowser(browser string) string {
	browser = strings.TrimSpace(browser)
	if browser == "" {
		return "chrome"
	}
	return browser
}
