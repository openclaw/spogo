---
title: Queue
description: "Add tracks to and inspect the playback queue."
---

# Queue

The queue is the up-next list managed by Spotify Connect. It survives device transfers and persists across pause/resume.

## queue add

```bash
spogo queue add <id|url>
```

Appends one track to the queue. Accepts a track URI, URL, or bare ID:

```bash
spogo queue add spotify:track:7hQJA50XrCWABAu5v6QZ4i
spogo queue add https://open.spotify.com/track/4PTG3Z6ehGkBFwjybzWkR8
spogo queue add 0sf12qNH5qcw8qpgymFOqD
```

`queue add` requires an active device. Start playback on a phone/desktop, or transfer to an available device with `spogo device set <name|id>` first. Connect queue additions do not resolve the global `--device` selector.

## queue show

```bash
spogo queue show
spogo queue show --plain
spogo queue show --json
```

Human and JSON output include the currently-playing item. Plain mode emits only upcoming tracks, with columns `type`, `ID`, `name`, `artists`, `album`, `URI`:

```
track   track-id   Track Name   Artist Name   Album Name   spotify:track:track-id
```

JSON mode includes `currently_playing` and a `queue` array with full track objects.

## queue clear

Spotify's API does not currently expose a way to clear the queue programmatically. The cleanest workaround is to start a new context, which replaces the queue:

```bash
spogo play spotify:track:7hQJA50XrCWABAu5v6QZ4i      # any single track
```

## Patterns

### Queue up the top results of a search

```bash
spogo search track "miles davis" --limit 5 --json |
  jq -r '.items[].uri' |
  while IFS= read -r uri; do spogo queue add "$uri"; done
```

### Queue a page of playlist tracks

`queue add` takes one track. To queue a page of playlist tracks (listings are capped at 50; use `--offset` for further pages):

```bash
spogo playlist tracks spotify:playlist:37i9dQZF1DXcBWIGoYBM5M --json |
  jq -r '.items[].uri' |
  while IFS= read -r uri; do spogo queue add "$uri"; done
```

For long playlists this is N HTTP calls — usually faster to just `play` the playlist as a context.
