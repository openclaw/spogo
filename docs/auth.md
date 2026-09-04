---
title: Auth
description: "Authenticate spogo with browser cookies or Spotify Authorization Code OAuth with PKCE."
---

# Auth

spogo supports two authentication mechanisms:

- **Browser cookies** are the default and preserve the existing Connect/internal-endpoint behavior.
- **Spotify OAuth** uses the official Authorization Code flow with PKCE for Web API requests. It does not use or store a client secret.

Choose Web API authentication with `--auth cookies|oauth`, `SPOGO_AUTH`, or the profile's `auth` setting. The default is `cookies`, so existing profiles keep their current behavior.

## Which auth should I use?

| Goal | Engine | Auth | Credentials required |
| --- | --- | --- | --- |
| Existing internal endpoints and Connect playback | `connect` or `auto` | `cookies` | Spotify browser cookies |
| Official Spotify Web API only | `web` | `oauth` | OAuth token cache + Spotify client ID |
| Connect first, official Web API for public-API fallbacks | `connect` or `auto` | `oauth` | Both browser cookies and OAuth |

OAuth cannot replace cookies for Spotify's internal Connect protocol. If you want a cookie-free setup, use `--engine web --auth oauth`.

## Official Spotify OAuth

spogo implements Spotify's [Authorization Code with PKCE flow](https://developer.spotify.com/documentation/web-api/tutorials/code-pkce-flow). PKCE is designed for installed/public clients that cannot safely hold a client secret.

### 1. Register a Spotify application

Create an application in the Spotify Developer Dashboard and add this redirect URI exactly:

```text
http://127.0.0.1:8888/callback
```

Spotify requires an explicit loopback IP. `localhost` is not accepted. A different loopback URI is fine, but it must match the dashboard entry and the value passed to spogo exactly.

### 2. Log in

```bash
spogo auth oauth login --client-id YOUR_SPOTIFY_CLIENT_ID
```

spogo starts a loopback-only callback server, generates a cryptographically random state value and PKCE verifier, opens the Spotify authorization page, validates the callback state, exchanges the code, and stores the resulting access and refresh tokens.

If the machine cannot open a browser automatically:

```bash
spogo auth oauth login --client-id YOUR_SPOTIFY_CLIENT_ID --no-open
```

Open the URL printed to stderr. The command waits up to five minutes by default; change that with `--wait-timeout`.

For a custom registered callback:

```bash
spogo auth oauth login \
  --client-id YOUR_SPOTIFY_CLIENT_ID \
  --redirect-uri http://127.0.0.1:9999/callback
```

The login stores only the non-secret client ID, redirect URI, and `auth = "oauth"` in the profile config. **Client secrets are unsupported and are never read from or written to config.**

### 3. Use the Web API client

```bash
spogo --engine web --auth oauth search track "weezer" --limit 5
spogo --engine web --auth oauth library tracks list --limit 20
```

After login, `auth = "oauth"` is saved for the profile, so `--auth oauth` is optional. `--engine web` remains explicit because the default `connect` engine still requires cookies.

### OAuth scopes

spogo requests the scopes needed by its existing Web API surface:

- playback state read/write and currently-playing access
- saved-library read/write
- followed-artist read/write
- private/collaborative playlist read and public/private playlist write
- top tracks and recently played history
- private account data required by Spotify search/profile endpoints

It does not request email access, streaming, image upload, or Web Playback SDK scopes.

### OAuth status and clearing

```bash
spogo auth oauth status
spogo auth oauth clear
```

`status` reads only local metadata. It never prints access or refresh token values and does not call Spotify. `clear` removes the token cache and returns the profile to the default cookie auth selection while retaining the non-secret client ID and redirect URI for a future login.

Access tokens refresh automatically before expiry. Spotify refresh responses that omit a replacement refresh token retain the previous refresh token. If Spotify rejects or expires the refresh token, run `auth oauth login` again.

### OAuth config and environment

Profile config example:

```toml
[profile.default]
auth = "oauth"
spotify_client_id = "your-public-client-id"
spotify_redirect_uri = "http://127.0.0.1:8888/callback"
engine = "web"
```

Equivalent environment overrides:

```bash
export SPOGO_AUTH=oauth
export SPOGO_SPOTIFY_CLIENT_ID=your-public-client-id
export SPOGO_SPOTIFY_REDIRECT_URI=http://127.0.0.1:8888/callback
export SPOGO_ENGINE=web
```

There is intentionally no client-secret config key, flag, or environment variable.

### OAuth token storage

OAuth tokens are stored per profile under the config directory:

```text
<config-dir>/spogo/oauth/<profile>.json
```

Profile names that are not portable lowercase filename segments are encoded before deriving the token and lock filenames, so separators, traversal components, Windows-reserved names, and case variants cannot escape or alias within the OAuth directory.

The OAuth directory is mode `0700` and token file is mode `0600` on POSIX systems. Writes use a same-directory temporary file, file sync, and atomic rename. spogo refuses to load a token file that is readable or writable by group/other users.

Treat the token cache as a credential. Do not copy it into source control, logs, shell history, or CI artifacts.

## Browser cookie auth

Cookie auth remains the default. spogo reads the cookies your browser already has for `open.spotify.com` and uses them for the internal Web Player and Connect protocols. The cookie machinery comes from [steipete/sweetcookie](https://github.com/steipete/sweetcookie).

### What spogo needs

- `sp_dc` is required.
- `sp_key` is optional and helps with rotation.
- `sp_t` is recommended for Connect playback control.

### Importing from a browser

```bash
spogo auth import --browser chrome
```

Supported browser names are `chrome`, `brave`, `edge`, `firefox`, and `safari`.

For a non-default browser profile:

```bash
spogo auth import --browser chrome --browser-profile "Profile 1"
```

For a specific cookie store file:

```bash
spogo auth import --cookie-path /path/to/cookies.sqlite
```

When the browser-store read returns nothing, spogo surfaces the underlying warning, such as a locked keychain, missing profile, or decryption failure.

### Manual paste (WSL fallback)

1. In Chrome, open DevTools, then Application, Cookies, `https://open.spotify.com`.
2. Copy `sp_dc` and, preferably, `sp_t`. `sp_key` is optional.
3. Run:

```bash
spogo auth paste
```

For non-interactive input:

```bash
printf '%s\n%s\n' "sp_dc=..." "sp_t=..." | spogo auth paste --no-input
```

### Cookie status and clearing

```bash
spogo auth status
spogo auth clear
```

These existing commands remain cookie-specific. They do not inspect or clear OAuth tokens.

`auth status` does not call Spotify. To verify cookies actually work, run a command that uses them:

```bash
spogo status
spogo search track "test" --limit 1
```

## Multiple accounts

Profiles keep cookie jars, OAuth token caches, and settings separate:

```bash
spogo --profile work auth import --browser chrome --browser-profile "Profile 1"
spogo --profile personal auth oauth login --client-id YOUR_SPOTIFY_CLIENT_ID

spogo --profile work status
spogo --profile personal --engine web search track "test"
```

## Troubleshooting

- **`no cookies found`**: choose the correct browser profile or use `auth paste`.
- **OAuth callback bind failure**: another process is using the registered port. Stop it or register and pass a different loopback redirect URI.
- **OAuth state mismatch**: leave the command running and use the newest authorization URL it printed. Invalid callbacks are rejected.
- **OAuth token cache permissions error**: change the token file to owner-only mode (`0600`) and its directory to `0700`, or clear and log in again.
- **OAuth works with `web` but `connect` fails**: expected. Connect still requires Spotify browser cookies.
- **OAuth refresh rejected**: run `spogo auth oauth login` again.
- **`401`/`403` with cookies**: re-import after logging back into `open.spotify.com`.

See [Troubleshooting](troubleshooting.md) for more.
