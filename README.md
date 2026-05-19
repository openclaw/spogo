# 🎧 spogo - Spotify, but make it terminal.

 Power CLI using web cookies. Search, control playback, manage library/playlists, and script with JSON/plain output.

## Features

- Search tracks, albums, artists, playlists, shows, episodes
- Playback control: play/pause/next/prev/seek/volume/shuffle/repeat
- Play artists (starts with top tracks)
- Queue management
- Library management (save/remove/follow)
- User listening data: top tracks by Spotify affinity and available recent plays
- Playlist management (create/add/remove/list)
- Device selection and status
- Browser cookie import via `sweetcookie`
- `--json` and `--plain` for scripting
- Colorized human output (respects `NO_COLOR`, `TERM=dumb`, `--no-color`)
- Engine switch: `auto` (connect → web), `connect` (internal endpoints), `web` (Web API endpoints; search/info/playback fall back to connect on rate limit)

## Why Cookies?

Spotify's official API has strict rate limits that make it impractical for agents and automation. By using browser cookies to authenticate with Spotify's internal web API (the same one their web player uses), spogo bypasses these limitations:

- **No rate limits** - Use the same endpoints as open.spotify.com
- **No app registration** - No need to create a Spotify Developer app
- **Full functionality** - Access to everything the web player can do
- **Agent-friendly** - Perfect for AI assistants and automation scripts

Import your cookies once with `sweetcookie` and you're good to go (defaults to Chrome).

## Install

### Homebrew

```bash
brew install steipete/tap/spogo
```

### Build from Source

```bash
go install github.com/steipete/spogo/cmd/spogo@latest
```

## Quick start

```bash
spogo auth import --browser chrome
spogo auth import --browser chrome --browser-profile "Profile 1"
spogo search track "weezer" --limit 5
spogo play spotify:track:7hQJA50XrCWABAu5v6QZ4i
spogo status
```

## Usage

```bash
spogo [global flags] <command> [args]
```

Global flags:

- `--config <path>` config file path
- `--profile <name>` profile name
- `--timeout <dur>` request timeout (default 10s)
- `--market <cc>` market country code
- `--language <tag>` language/locale (default `en`)
- `--device <name|id>` target device
- `--engine <auto|web|connect|applescript>` API engine (default `connect`, `applescript` is macOS-only)
- `--json` / `--plain`
- `--no-color`
- `-q, --quiet` / `-v, --verbose` / `-d, --debug`

Env overrides:

- `SPOGO_TOTP_SECRET_URL` override TOTP secret source (http(s) or `file://...`)
- `SPOGO_CONNECT_VERSION` override connect client version sent to playback endpoints

Commands:

- `auth status|import|paste|clear`
- `search track|album|artist|playlist|show|episode`
- `track info`, `album info`, `artist info`, `playlist info`, `show info`, `episode info`
- `play [<id|url>] [--type ...] [--shuffle]`, `pause`, `next`, `prev`, `seek`, `volume`, `shuffle`, `repeat`, `status`
- `queue add|show`
- `library tracks|albums|artists|playlists`
- `user top-tracks|history`
- `playlist create|add|remove|tracks`
- `device list|set`

Full spec: `docs/spec.md`.

## Cookies

`spogo` uses browser cookies (via `sweetcookie`) to fetch a web access token. Import cookies once:

```bash
spogo auth import --browser chrome
```

Defaults: Chrome + Default profile. Cookies are stored under your config directory (per profile).
If import still fails, `spogo` now surfaces browser-store warnings instead of only printing `no cookies found`.

### Manual cookie paste (WSL fallback)

If WSL cookie import/decryption is broken, paste cookies from Chrome DevTools:

1) Developer Tools -> Application tab -> Cookies -> `https://open.spotify.com`
2) Copy `sp_dc` (required), `sp_key` (optional), `sp_t` (recommended for connect playback)
3) Run:

```bash
spogo auth paste
```

Non-interactive:

```bash
printf '%s\n%s\n' "sp_dc=..." "sp_t=..." | spogo auth paste --no-input
```

## Auto engine notes

- `auto` tries connect first, then falls back to web on unsupported features or rate limits.

## Connect engine notes

- `connect` uses Spotify's internal connect-state endpoints for playback control.
- Auth/session data and the last active playback route are cached per profile so repeated playback commands avoid a full Connect state refresh when the route is still valid.
- Search/info prefer the internal GraphQL API and fall back to web search if hashes can’t be resolved.

## Web engine notes

- `web` uses Web API endpoints; search/info/playback auto-fallback to connect when rate limited.

## Output

- Human output by default
- `--plain` for line-oriented output
- `--json` for structured output

## Legal

This tool interacts with Spotify's web endpoints. Use responsibly and in accordance with Spotify's Terms of Service.

## License

MIT
