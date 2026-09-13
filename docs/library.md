---
title: Library & Playlists
description: "List, save, remove tracks/albums/artists, and create or mutate playlists from the terminal."
---

# Library & Playlists

Your saved tracks, albums, followed artists, and playlists — all listable, mutable, and pipeable.

## library tracks

```bash
spogo library tracks list [--limit N]
spogo library tracks add <id|url...>
spogo library tracks remove <id|url...>
```

`add`/`remove` accept multiple IDs or URLs in one call:

```bash
spogo library tracks add \
  spotify:track:7hQJA50XrCWABAu5v6QZ4i \
  https://open.spotify.com/track/4PTG3Z6ehGkBFwjybzWkR8
```

## library albums

```bash
spogo library albums list [--limit N]
spogo library albums add <id|url...>
spogo library albums remove <id|url...>
```

## library artists

```bash
spogo library artists list [--limit N] [--after <artist-id>]
spogo library artists follow <id|url...>
spogo library artists unfollow <id|url...>
```

`--after` paginates by artist ID — pass the last ID from the previous page to fetch the next.

Listing followed artists uses Spotify's internal web-player library operation. Following or unfollowing an artist still uses the public Web API and may return a rate-limit cooldown.

## library playlists

```bash
spogo library playlists list [--limit N]
```

Lists one page of playlists you own or follow; use `--offset` to fetch further pages (maximum `--limit` is 50). To list **tracks** in a playlist, use `playlist tracks` below.

Playlist and library collection listings use the internal web-player API. Saving/removing tracks or albums and creating playlists still require the public Web API, so those mutations may be rate-limited.

## playlist follow / unfollow / following

```bash
spogo playlist follow <id|uri|url>
spogo playlist following <id|uri|url> --plain
spogo playlist unfollow <id|uri|url>
```

These commands save a playlist to your library, check its membership, or remove it from your library. They accept a playlist ID, Spotify URI, or URL and do not change the playlist's visibility or contents.

They use Spotify's public Web API library endpoints with the selected cookie or OAuth authentication, including when the engine is `connect`. A rate-limit cooldown can apply; switching engines does not bypass it. Cookie-free access uses `--engine web --auth oauth` after `auth oauth login`.

`following --plain` prints `true` or `false`; `--json` emits `{"following":true}` or `{"following":false}`. Both membership states exit successfully. Follow/unfollow print `ok` in plain mode and emit `{"id":"...","status":"ok"}` in JSON mode. Errors use a nonzero exit code without a success payload.

## playlist create

```bash
spogo playlist create "Road Trip"
spogo playlist create "Team Mix" --public
spogo playlist create "Shared Notes" --collab
```

`--public` marks the playlist as discoverable; `--collab` makes it editable by collaborators (collaborative playlists must be private).

## playlist add / remove

```bash
spogo playlist add <playlist> <track...>
spogo playlist remove <playlist> <track...>
```

`<playlist>` is a playlist ID, `spotify:playlist:...` URI, or `https://open.spotify.com/playlist/...` URL. Playlist names are not resolved. Tracks accept the same flexible forms as `library tracks add`.

```bash
spogo playlist add spotify:playlist:37i9dQZF1DXcBWIGoYBM5M \
  spotify:track:7hQJA50XrCWABAu5v6QZ4i \
  spotify:track:0sf12qNH5qcw8qpgymFOqD

spogo playlist remove 37i9dQZF1DXcBWIGoYBM5M spotify:track:7hQJA50XrCWABAu5v6QZ4i
```

Playlist mutations route through Connect by default — Connect avoids the Web API rate limits that bite when you script bulk add/remove. spogo automatically detects writable playlists and falls back to Web API where Connect can't help.

## playlist tracks

```bash
spogo playlist tracks <playlist> [--limit N]
```

Lists the items inside a playlist in order, including repeated tracks. Library listings still deduplicate entities; playlist positions are preserved:

```bash
spogo playlist tracks spotify:playlist:37i9dQZF1DXcBWIGoYBM5M --plain | head
spogo playlist tracks 37i9dQZF1DXcBWIGoYBM5M --json | jq '.items[].name'
```

## Common patterns

### Save the currently playing track

```bash
id=$(spogo status --json | jq -r '.item.id')
spogo library tracks add "$id"
```

### Build a playlist from a search

```bash
playlist_id=$(spogo playlist create "Lo-Fi Coding" --json | jq -r .id)
spogo search track "lo-fi" --limit 20 --json |
  jq -r '.items[].uri' |
  while IFS= read -r uri; do spogo playlist add "$playlist_id" "$uri"; done
```

### Snapshot a page of liked tracks to a file

```bash
spogo library tracks list --limit 50 --offset 0 --json > liked-tracks-page.json
```

Listings are paginated: `--limit` is capped at 50, and `--offset` selects the next page. A large limit does not automatically fetch the whole library.

## Errors

- **`playlist not found`** — pass the playlist ID, URI, or URL.
- **`not collaborative`** — only owners and explicitly added collaborators can mutate a playlist.
- **`429 too many requests`** — honor the retry-after hint. Playlist membership operations use the Web API in every engine, so switching engines does not bypass their cooldown.

See [Engines](engines.md) and [Output](output.md) for output and engine details.
