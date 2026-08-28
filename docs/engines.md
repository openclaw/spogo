---
title: Engines
description: "Choose between connect, web, auto, and applescript — what each engine talks to and when to use it."
---

# Engines

spogo can talk to Spotify through one of four engines. Pick whichever matches what you need, or let `auto` decide.

## Quick pick

| Need | Engine |
| --- | --- |
| Default; works for almost everything | `connect` |
| Account where Connect is unavailable | `web` |
| "I just want it to work" | `auto` |
| Drive Spotify.app on macOS, no network needed | `applescript` |

Set with `--engine <name>` per call, or globally with `SPOGO_ENGINE`. Engine selection is separate from Web API authentication, selected with `--auth cookies|oauth` or `SPOGO_AUTH`.

## connect (default)

Talks to Spotify's internal Connect endpoints — the same ones the official desktop and mobile apps use to coordinate playback across devices. spogo's first choice for everything.

**Best for**

- Playback control (play, pause, next, prev, seek, volume, shuffle, repeat).
- Device discovery and transfer.
- Playlist mutations under heavy use (Connect doesn't hit the Web API rate limits).
- Search and item info via the internal GraphQL surface, including episode lookup.
- Listing followed artists, saved albums/tracks, and playlists; user top tracks and recently played history also use internal endpoints.

**Authentication**

Connect always requires Spotify browser cookies. When Connect delegates an operation to the public Web API, that fallback uses the selected `--auth cookies|oauth` provider. Therefore `--engine connect --auth oauth` requires both cookies for Connect and an OAuth login for Web API fallbacks.

**Tradeoffs**

- Saving/removing library tracks or albums, following/unfollowing artists, creating playlists, and artist-top-track lookups used by artist playback still require the public Web API.
- Transfers without a Connect origin device, some hardware volume/playback requests, and failed internal catalog/library lookups may also fall back to the public Web API.
- These public-API paths can be rate-limited even with `--engine connect`; a `429` includes Spotify's `retry-after hint` whenever one is supplied.

## web

The public Spotify Web API. Slower, lower throughput, and rate-limited according to Spotify's account- and application-specific policies; cookie-derived tokens can encounter aggressive cooldowns, including retry hints measured in hours.

**Authentication**

Use the existing cookie-derived Web API token with `--auth cookies` (the default), or the official Authorization Code with PKCE token with `--auth oauth`. A cookie-free setup is:

```bash
spogo auth oauth login --client-id YOUR_SPOTIFY_CLIENT_ID
spogo --engine web --auth oauth search track "weezer"
```

**Best for**

- Accounts that can't use Connect (rare — usually corporate or family-restricted).
- Forcing the documented public API for a reproducible test.
- Anything that requires Web API specific endpoints not yet in Connect.

**Tradeoffs**

- Rate limits can apply even without bulk operations. If you see `429`, prefer `connect` for supported reads and honor the reported `retry-after hint`; Web-API-only operations cannot bypass that cooldown by switching engines.
- Search/info/playback auto-fall-back to Connect when rate limited, so practical behavior is closer to `auto`.

## auto

Try `connect` first, then fall back to `web` for unsupported features or rate limits. Because Connect is first, `auto` still requires browser cookies even when `--auth oauth` selects OAuth for the Web API fallback. On macOS, playback status and controls get one final fallback to the already-local Spotify.app through AppleScript after both remote engines fail, including when cookies are missing.

```bash
spogo --engine auto play spotify:playlist:...
```

Most users don't need this — `connect` already falls back to web for the specific paths where it has to. `auto` is useful when you want **explicit** fallback behavior across all calls.

The AppleScript last resort is limited to `status`, `play`, `pause`, `next`, `prev`, `seek`, `volume`, `shuffle`, and `repeat`. Search, catalog lookup, library operations, playlists, queue commands, and devices remain remote-only; explicit `--engine connect` and `--engine web` never use AppleScript.

## applescript (macOS only)

Drives the local Spotify desktop app via AppleScript. No network, no cookies, no rate limits — but only the Mac you're on can be controlled, and you only see the local app's view (no Connect device list).

```bash
spogo --engine applescript play
spogo --engine applescript pause
spogo --engine applescript next
spogo --engine applescript status
```

**Best for**

- Quick local hotkeys / shortcuts (Raycast, Alfred, sketchybar, etc.) where network round trips are wasted.
- Sandboxed environments where cookie auth is awkward.
- Scripts that just need "pause my Mac's Spotify" without touching cloud state.

**Tradeoffs**

- macOS only.
- No Connect device list (`device list` shows just the Mac), no transfers.
- Search, catalog lookups, library, and playlists are not AppleScript operations; explicit `applescript` may delegate them to its configured remote fallback.

## Setting an engine

Per command:

```bash
spogo --engine connect play
spogo --engine web --auth oauth search track "weezer"
spogo --engine applescript pause
```

Per shell:

```bash
export SPOGO_ENGINE=web
export SPOGO_AUTH=oauth
```

In a config profile (`~/.config/spogo/<profile>/config.toml` or platform equivalent):

```toml
engine = "web"
auth = "oauth"
```

## Diagnosing engine issues

```bash
spogo --debug status
```

Debug logging on stderr shows which engine handled each call and any fallbacks that fired. See [Output](output.md) and [Troubleshooting](troubleshooting.md).
