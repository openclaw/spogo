# spogo CLI spec (v0.10.0)

One-liner: Spotify power CLI using web cookies; search + playback control.
Parser: Kong.
Cookies: steipete/sweetcookie (local sweetcookie).
Output: human by default; `--plain` or `--json`.
Color: on by default; respects `NO_COLOR`, `TERM=dumb`, `--no-color`.
Platforms: macOS, Linux, Windows.

## Usage

```
spogo [global flags] <command> [args]
```

## Global flags

- `-h, --help`
- `--version`
- `-q, --quiet`
- `-v, --verbose`
- `-d, --debug`
- `--json`
- `--plain`
- `--no-color`
- `--config <path>` default: `os.UserConfigDir()/spogo/config.toml`
- `--profile <name>` default: `default`
- `--timeout <dur>` default: `10s`
- `--market <cc>` default: account market or `US`
- `--language <tag>` default: `en`
- `--device <name|id>` default: active device
- `--engine <auto|web|connect|applescript>` default: `connect` (`applescript` is macOS-only)
- `--no-input`

## Commands

### completion

- `spogo completion fish`
  - prints fish shell completions to stdout
  - does not load Spotify auth, cookies, or profile configuration

### auth

- `spogo auth status`
- `spogo auth import`
  - flags: `--browser <chrome|brave|edge|firefox|safari>` default: `chrome`
  - `--browser-profile <name>`
  - `--cookie-path <file>`
  - `--domain <host>` default `spotify.com`
  - when browser reads fail, surface underlying browser-store warnings
- `spogo auth paste`
  - reads cookie values from stdin (prompts when interactive)
  - `--cookie-path <file>`
  - `--domain <suffix>` default `spotify.com`
  - `--path <path>` default `/`
- `spogo auth clear`

### search

- `spogo search track <query> [--limit N] [--offset N]`
- `spogo search album <query> [--limit N] [--offset N]`
- `spogo search artist <query> [--limit N] [--offset N]`
- `spogo search playlist <query> [--limit N] [--offset N]`
- `spogo search episode <query> [--limit N] [--offset N]`
- `spogo search show <query> [--limit N] [--offset N]`

### info

- `spogo track info <id|url>`
- `spogo album info <id|url>`
- `spogo artist info <id|url>`
- `spogo playlist info <id|url>`
- `spogo show info <id|url>`
- `spogo episode info <id|url>`

### playback

- `spogo play [<id|url>]` (track/album/playlist/show)
  - optional: `--type <track|album|playlist|show|episode>` for raw IDs
  - optional: `--shuffle` enable shuffle before playing (randomizes first track for context URIs)
  - artist URIs play top tracks (starts with the first)
- `spogo pause`
- `spogo next`
- `spogo prev`
- `spogo seek <ms|mm:ss>`
- `spogo volume <0-100>`
- `spogo shuffle <on|off>`
- `spogo repeat <off|track|context>`
- `spogo status`

### queue

- `spogo queue add <id|url>`
- `spogo queue show`
- `spogo queue clear` (not supported by Spotify API yet)

### library

- `spogo library tracks list [--limit N]`
- `spogo library tracks add <id|url...>`
- `spogo library tracks remove <id|url...>`
- `spogo library albums list [--limit N]`
- `spogo library albums add <id|url...>`
- `spogo library albums remove <id|url...>`
- `spogo library artists list [--limit N] [--after <artist-id>]`
- `spogo library artists follow <id|url...>`
- `spogo library artists unfollow <id|url...>`
- `spogo library playlists list [--limit N]`

### user

- `spogo user top-tracks [--period long_term|medium_term|short_term] [--limit N] [--offset N]`
  - returns Spotify affinity-ranked tracks, not play counts
  - `long_term` = years of listening data; `medium_term` = about 6 months; `short_term` = about 4 weeks
- `spogo user history [--period long_term|medium_term|short_term] [--limit N] [--after <ms>] [--before <ms>]`
  - returns retained recent plays available from Spotify, not a complete listening archive
  - paginates backward with `before` cursors until it reaches `--limit`, the selected period, or Spotify's retained history; client cap is 200 items
  - `medium_term`, `short_term`, and `--after` are local lower-bound filters

### playlists

- `spogo playlist create <name> [--public] [--collab]`
- `spogo playlist add <playlist> <track...>`
- `spogo playlist remove <playlist> <track...>`
- `spogo playlist tracks <playlist> [--limit N]`

### devices

- `spogo device list`
- `spogo device set <name|id>`
  - falls back to Web API transfer when Connect state has no origin device

## Output contract

- stdout: primary results; human or machine modes.
- stderr: warnings/errors/logs.
- `--plain`: stable, line-oriented, tab-separated fields.
- `--json`: stable, documented keys per command.

## Engines

- `auto`: connect first; fall back to web for unsupported features or rate limits.
- `connect`: internal connect-state endpoints for playback; GraphQL for search/info. Auth/session data and the last active playback route are cached per profile.
- `web`: Web API endpoints; search/info/playback auto-fallback to connect when rate limited.

## Exit codes

- `0` success
- `1` generic failure
- `2` invalid usage/validation
- `3` auth/cookies missing or invalid
- `4` network/timeouts

## Config / env

- Env prefix: `SPOGO_`
- Precedence: flags > env > config
- Secrets: never via flags; use browser cookies only.
- Overrides:
  - `SPOGO_TOTP_SECRET_URL` (http(s) or `file://...`)
  - `SPOGO_CONNECT_VERSION` (connect playback client version)

## Examples

- `spogo auth import --browser chrome`
- `spogo search track "weezer" --limit 5 --plain`
- `spogo play spotify:track:7hQJA50XrCWABAu5v6QZ4i`
- `spogo device list --json`
- `spogo playlist create "Road Trip" --public`
