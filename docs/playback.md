---
title: Playback
description: "Play, pause, seek, volume, shuffle, repeat — drive Spotify playback from the terminal."
---

# Playback

All playback commands act on the currently active Spotify Connect device unless you pass `--device`. Use [`spogo device list`](devices.md) to see what's available and `spogo device set <name|id>` to switch.

## play

```bash
spogo play [<id|url>] [--type <track|album|playlist|show|episode>] [--shuffle]
```

Accepts:

- A Spotify URI: `spotify:track:7hQJA50XrCWABAu5v6QZ4i`, `spotify:album:...`, `spotify:playlist:...`, `spotify:show:...`, `spotify:episode:...`, `spotify:artist:...`.
- A web URL: `https://open.spotify.com/track/7hQJA50XrCWABAu5v6QZ4i`.
- A bare ID — combine with `--type` to disambiguate.
- No argument — resumes the current item.

Behavior:

- **Tracks** start immediately.
- **Albums / playlists / shows** start a context — `spogo next` walks through items.
- **Artists** start with the artist's top tracks (first track first).
- `--shuffle` enables shuffle on the device before play, randomizing the first track for context URIs.

Examples:

```bash
spogo play                                                      # resume
spogo play spotify:track:7hQJA50XrCWABAu5v6QZ4i                 # one track
spogo play https://open.spotify.com/album/4aawyAB9vmqN3uQ7FjRGTy
spogo play 37i9dQZF1DXcBWIGoYBM5M --type playlist               # bare ID
spogo play spotify:playlist:37i9dQZF1DXcBWIGoYBM5M --shuffle    # shuffle on
spogo play spotify:artist:6sFIWsNpZYqfjUpaCgueju                # top tracks
```

## pause / resume

```bash
spogo pause
spogo play              # no argument resumes
```

## next / prev

```bash
spogo next
spogo prev              # restart current track if past ~3s, else previous
```

## seek

```bash
spogo seek 90000        # milliseconds
spogo seek 1:30         # mm:ss
spogo seek 0            # back to start
```

## volume

```bash
spogo volume 60         # 0-100
spogo volume 0          # mute
```

Some devices ignore volume changes (e.g. Spotify Connect on hardware that exposes its own volume).

## shuffle / repeat

```bash
spogo shuffle on
spogo shuffle off

spogo repeat off
spogo repeat track       # loop current track
spogo repeat context     # loop current album/playlist
```

## status

```bash
spogo status             # human, color
spogo status --plain     # tab-separated key value
spogo status --json      # full payload
```

JSON shape includes `is_playing`, `progress_ms`, `device`, `item` (track or episode), `context`, `repeat_state`, `shuffle_state`. Use `jq` to pluck what you need:

```bash
spogo status --json | jq -r '.item.name + " — " + (.item.artists|map(.name)|join(", "))'
```

## Targeting a specific device

Every playback command accepts `--device <name|id>`:

```bash
spogo play spotify:track:... --device "Kitchen"
spogo volume 30 --device "MacBook Pro"
spogo pause --device 0d1841b0976bae2a3a310dd74c0f3df354899bc8
```

When Connect state has no origin device, `spogo` falls back to the Web API transfer endpoint instead of failing.

## Engine notes

- **`connect`** (default) — playback control via Spotify's internal Connect endpoints. Best fidelity for transitions, queueing, and device transfer; auto-falls-back to Web API for transfers when no origin device exists.
- **`web`** — the public Web API. Slower, rate-limited, but the only option for accounts with restrictive Connect availability.
- **`auto`** — Connect first, then Web; on macOS, playback status/control finally fall back to Spotify.app through AppleScript if both remote engines fail.
- **`applescript`** (macOS only) — drive Spotify.app directly via AppleScript. No network, but only sees the local Mac app.

See [Engines](engines.md) for the full breakdown.

## Errors

- `no active device` — open Spotify on a phone/desktop/Connect speaker first, or pass `--device`.
- `403 PREMIUM_REQUIRED` — playback requires a Spotify Premium account.
- `429 too many requests` — prefer `connect` for supported playback; Web-API-only fallback paths can still be limited, so honor the reported `retry-after hint`.
