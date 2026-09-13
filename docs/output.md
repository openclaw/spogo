---
title: Output
description: "JSON and tab-separated output, color, stdout/stderr, and exit codes."
---

# Output

Use `--json` for structured data and `--plain` for tab-separated rows. These flags are mutually exclusive. Command errors go to stderr and return a nonzero exit code.

## Human output and color

Human output is the default. Color requires a terminal on stdout and is disabled by `--no-color`, a nonempty `NO_COLOR`, or `TERM=dumb`. Passing `--no-color=false` does not force color into a pipe.

`--quiet` suppresses human result output. It does not suppress JSON/plain results or errors. `--verbose` and `--debug` are accepted for compatibility, but currently do not enable additional HTTP or engine tracing.

## Plain output

Rows have no header. Field order is stable:

| Result | Columns, in order |
| --- | --- |
| Track | type (`track`), ID, name, artists, album, URI |
| Album | type (`album`), ID, name, artists, release date, track count |
| Artist | type (`artist`), ID, name, followers |
| Playlist | type (`playlist`), ID, name, owner, track count |
| Show | type (`show`), ID, name, publisher, episode count |
| Episode | type (`episode`), ID, name, duration in milliseconds |
| Playback status | is playing, progress in milliseconds, device name, item name |
| Device | ID, name, is active |
| Top track | rank, then the track columns above |
| History item | played-at timestamp, then the track columns above |

`queue show --plain` prints upcoming tracks only. JSON includes the current item separately. Successful mutation commands generally print `ok`; commands with additional results are described in their command guides.

Artists are joined with a comma and space. Text fields currently pass tabs and newlines through unchanged; use JSON when arbitrary metadata must round-trip safely.

```bash
# Track URIs are column six, not column one.
spogo search track "weezer" --limit 3 --plain | cut -f6
```

## JSON output

Search returns an object with `type`, `limit`, `offset`, `total`, and `items`. Library and playlist-track listings use `total` and `items`. Item lookups return a single item. Item `artists` is an array of strings.

Playback status includes `is_playing`, `progress_ms`, `device`, `shuffle`, and `repeat`, plus `item` when available. Queue output uses `currently_playing` when available and `queue` for upcoming items. Device listing returns an array.

```bash
spogo status --json | jq -r '.item.name + " — " + (.item.artists | join(", "))'
spogo playlist tracks spotify:playlist:37i9dQZF1DXcBWIGoYBM5M --json | jq '.items[].name'
```

Fields may be added; existing keys are not renamed or removed without a major version bump. Optional metadata may be absent. Consumers should tolerate `null` for empty collections and missing optional fields.

## Prompts and diagnostics

Cookie paste accepts piped values without prompting. `--no-input` refuses interactive cookie paste when stdin is a terminal. OAuth login still prints an authorization URL and waits for a browser callback; it is not a headless login flow.

```bash
spogo auth paste --no-input < cookies.txt
spogo status --json > status.json 2> spogo.log
```

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Generic failure |
| `2` | Parser usage or global-option validation error |
| `3` | Missing browser cookies, OAuth authentication failure, or Spotify HTTP 401/403 |
| `4` | Network / timeout |

Some command-specific validation errors currently return `1`. Cookie status is a local inventory check, not proof that the session is valid on Spotify.

Capture a command's status before applying shell negation:

```bash
if spogo status --json > status.json; then
  jq '.item' status.json
else
  code=$?
  case "$code" in
    3) echo "Authentication failed" >&2 ;;
    4) echo "Network request failed" >&2 ;;
    *) echo "spogo failed (exit $code)" >&2 ;;
  esac
  exit "$code"
fi
```
