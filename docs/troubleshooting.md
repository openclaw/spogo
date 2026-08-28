---
title: Troubleshooting
description: "Common spogo failures — auth, devices, rate limits, WSL — and how to fix them."
---

# Troubleshooting

The first move when anything misbehaves is to re-run with `-d`:

```bash
spogo -d <command>
```

Debug logs go to stderr and include engine choices, fallbacks, HTTP status codes, and request IDs. They will not pollute a `--json` or `--plain` pipeline.

## Auth

### `no cookies found`

spogo couldn't read cookies from the browser store. Check:

1. You're actually logged in to `https://open.spotify.com` in that browser.
2. You picked the right `--browser-profile` (Chrome's default is `Default`, but most users have `Profile 1`, `Profile 2`).
3. spogo printed an underlying warning — recent versions surface the real reason (locked keychain, decryption failure, missing profile dir).

If browser-store reads keep failing, fall back to `auth paste` (see [Auth](auth.md#manual-paste-wsl-fallback)).

### `401 Unauthorized` / `403 Forbidden` from any read command

Cookies are stale. Re-import:

```bash
spogo auth import --browser chrome
spogo auth status
```

If that doesn't help, your browser's session may have expired. Visit `https://open.spotify.com`, log back in, then re-import.

### OAuth login says the callback address is already in use

The registered loopback port is occupied. Close the process using it, or add a different loopback redirect URI to the Spotify application and pass the exact same URI to `auth oauth login --redirect-uri`.

### OAuth works with `--engine web` but Connect fails

This is expected when cookies are missing. OAuth authenticates the public Web API client only. The internal `connect` protocol and the Connect-first `auto` engine still require Spotify browser cookies.

For OAuth without cookies:

```bash
spogo --engine web --auth oauth status
```

### OAuth token cache permissions are rejected

spogo requires owner-only OAuth credentials. On POSIX systems the token file must be `0600` and its directory must be `0700`. Fix those permissions, or run `spogo auth oauth clear` and log in again.

### OAuth refresh is rejected

The refresh token may have been revoked or expired. Run:

```bash
spogo auth oauth login
```

The client ID and redirect URI remain in the profile after `auth oauth clear`, so a client ID flag is only required when it is not already configured.

### macOS Chrome keychain prompt

The first cookie import will trigger a "Chrome wants to use your confidential information from your keychain" dialog. Click **Always Allow**. If you mis-click **Deny**, fix it via:

`Keychain Access → login → Passwords → search "Chrome Safe Storage" → right-click → Get Info → Access Control → +` add `spogo`.

### WSL: cookie decryption fails

Chrome on WSL has a fragile DPAPI/Linux-keyring combo. Use `auth paste` instead:

1. DevTools → Application → Cookies → `https://open.spotify.com` → copy `sp_dc`, `sp_t`.
2. `spogo auth paste`.

## Playback

### `no active device`

Open Spotify on a phone, desktop, or Connect speaker once so it registers, or pass `--device <name|id>`:

```bash
spogo device list
spogo play spotify:track:... --device "Kitchen"
```

### `403 PREMIUM_REQUIRED`

Playback control (play, pause, seek, transfer, queue) requires a Spotify Premium account. spogo's read-only commands (`status`, `search`, `library tracks list`, etc.) work on free accounts.

### Volume command does nothing

Some Connect endpoints expose their own hardware volume and ignore Spotify's volume command. Try `--device` to a different target, or set the volume on the device itself.

### `device set` fails with "Connect state has no origin device"

Recent spogo versions auto-fall-back to the Web API transfer endpoint here. If you're on an older version, upgrade:

```bash
brew upgrade spogo
```

## Search & info

### Empty results for a search that should match

The internal GraphQL search uses query hashes that occasionally roll. spogo falls back to web search when a hash can't be resolved; if both fail, try:

```bash
spogo --engine web search track "your query"
```

### `track info` returns sparse data

Older spogo versions had Connect responses missing artist/album for some track shapes. Upgrade to the latest release.

## Rate limits

### `429 too many requests`

Spotify's public Web API can rate-limit cookie-derived tokens aggressively, even when a command starts on the `connect` engine. If you see it:

- You're on `--engine web` — switch to `connect` or `auto` when the command has an internal equivalent.
- Saving/removing library tracks or albums, following/unfollowing artists, playlist creation, artist-top-track lookups, and some device/playback fallbacks still require the public API; changing engines cannot bypass their cooldown.
- Respect the `retry-after hint` printed with the error. Spotify can advertise cooldowns lasting hours or longer.
- You're hammering the API in a tight loop — add `sleep 0.2` between calls, or batch via `--limit`.
- Multiple spogo profiles are sharing the same Spotify account at the same time — the throttle is per-account, not per-process.

## Output

### Color codes leaking into a file

You're capturing stdout but spogo thinks stdout is a TTY. This shouldn't happen — spogo detects TTY correctly — but if it does, force off:

```bash
spogo --no-color status > out.txt
NO_COLOR=1 spogo status > out.txt
```

### `jq` complains the JSON is malformed

You're capturing stderr too. Redirect it:

```bash
spogo status --json 2>/dev/null | jq .
```

### Pipe is empty / silent

The command may be writing to stderr only (errors, prompts). Re-run without redirecting stderr to see what happened:

```bash
spogo <command>
```

## Engines

### "Connect engine returned X" — what does that mean?

Run with `-d` and look for the `engine=` line in the debug output. spogo logs which engine handled each call and any fallback that fired.

### AppleScript engine: "spotify not running"

Open Spotify.app first. AppleScript can't launch the app reliably; spogo expects it to already be open.

### AppleScript engine: search results differ from web

The Mac app uses local search. Switch to `connect` or `web` for canonical results.

## Diagnostics to share when filing an issue

```bash
spogo --version
spogo -d <failing command> 2> spogo.log
```

Attach `spogo.log` (redact any cookie values it contains) to a [GitHub issue](https://github.com/openclaw/spogo/issues).
