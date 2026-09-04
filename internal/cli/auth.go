package cli

import "time"

type AuthCmd struct {
	Status AuthStatusCmd `kong:"cmd,help='Show cookie status.'"`
	Import AuthImportCmd `kong:"cmd,help='Import browser cookies.'"`
	Paste  AuthPasteCmd  `kong:"cmd,help='Paste cookie values from the browser.'"`
	Clear  AuthClearCmd  `kong:"cmd,help='Clear stored cookies.'"`
	OAuth  AuthOAuthCmd  `kong:"cmd,name='oauth',help='Official Spotify OAuth for Web API requests.'"`
}

type AuthStatusCmd struct{}

type AuthImportCmd struct {
	Browser    string `help:"Browser name (chrome|brave|edge|firefox|safari)."`
	Profile    string `name:"browser-profile" help:"Browser profile name."`
	CookiePath string `help:"Cookie cache file path."`
	Domain     string `help:"Cookie domain suffix." default:"spotify.com"`
}

type AuthPasteCmd struct {
	CookiePath string `help:"Cookie cache file path."`
	Domain     string `help:"Cookie domain suffix." default:"spotify.com"`
	Path       string `help:"Cookie path." default:"/"`
}

type AuthClearCmd struct{}

type AuthOAuthCmd struct {
	Login  AuthOAuthLoginCmd  `kong:"cmd,help='Authorize with Spotify using Authorization Code with PKCE.'"`
	Status AuthOAuthStatusCmd `kong:"cmd,help='Show local OAuth token status.'"`
	Clear  AuthOAuthClearCmd  `kong:"cmd,help='Clear the local OAuth token cache.'"`
}

type AuthOAuthLoginCmd struct {
	ClientID    string        `name:"client-id" help:"Spotify application client ID."`
	RedirectURI string        `name:"redirect-uri" help:"Registered loopback redirect URI (default http://127.0.0.1:8888/callback)."`
	NoOpen      bool          `name:"no-open" help:"Print the authorization URL without opening a browser."`
	WaitTimeout time.Duration `name:"wait-timeout" help:"Maximum time to wait for the OAuth callback." default:"5m"`
}

type AuthOAuthStatusCmd struct{}

type AuthOAuthClearCmd struct{}

type authStatusPayload struct {
	CookieCount int    `json:"cookie_count"`
	HasSPDC     bool   `json:"has_sp_dc"`
	HasSPT      bool   `json:"has_sp_t"`
	HasSPKey    bool   `json:"has_sp_key"`
	Source      string `json:"source"`
}
