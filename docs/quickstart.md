---
title: Quickstart
description: "From install to playing your first track in five minutes — cookie import, search, play, status."
---

# Quickstart

Five minutes from `brew install` to controlling Spotify from the terminal.

## 1. Install

```bash
brew install steipete/tap/spogo
```

Other options live in [Install](install.md).

## 2. Import your browser cookies

spogo authenticates by reading the cookies your browser already has for `open.spotify.com`. Make sure you're logged in there in Chrome (or Brave, Edge, Firefox, Safari), then:

```bash
spogo auth import --browser chrome
```

Defaults: Chrome, the `Default` profile. To pick a non-default profile:

```bash
spogo auth import --browser chrome --browser-profile "Profile 1"
```

If something goes wrong (locked keychain, weird WSL setup), see [Auth](auth.md) for `auth paste` and other fallbacks.

For an official, cookie-free Web API setup instead, register `http://127.0.0.1:8888/callback` in a Spotify developer application and run:

```bash
spogo auth oauth login --client-id YOUR_SPOTIFY_CLIENT_ID
spogo --engine web --auth oauth search track "test" --limit 1
```

OAuth supports the Web API client. Connect and `auto` still require browser cookies.

Verify cookie auth:

```bash
spogo auth status
```

## 3. Find something to play

```bash
spogo search track "weezer say it ain't so" --limit 3
```

Add `--json` if you want structured output, or `--plain` for tab-separated lines.

## 4. Play it

```bash
spogo play spotify:track:7hQJA50XrCWABAu5v6QZ4i
```

You can also pass an `https://open.spotify.com/...` URL, or a playlist/album/show URI. spogo figures out the type from the URI; for raw IDs use `--type`.

## 5. Drive the rest

```bash
spogo status                 # what's playing
spogo pause
spogo next
spogo volume 60
spogo shuffle on
spogo queue add spotify:track:6rqhFgbbKwnb9MLmUQDhG6
spogo device list            # available speakers/players
spogo device set "Kitchen"   # switch playback there
```

## 6. Pipe it into something

```bash
# What track is playing right now, machine-readable?
spogo status --json | jq -r '.item.name'

# Save the top-5 search results into a playlist
spogo search track "lo-fi" --limit 5 --plain |
  awk '{print $1}' |
  xargs spogo playlist add "Lo-Fi Coding"
```

## Where to next

- [Auth](auth.md) — cookie import, official OAuth, token storage, and troubleshooting.
- [Engines](engines.md) — when to choose `connect`, `web`, `auto`, or `applescript`.
- [Output](output.md) — the JSON / plain contract.
- [Agents](agents.md) — end-to-end automation patterns.
- [Command Reference](commands.md) — every subcommand and flag.
