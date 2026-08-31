---
title: Command Reference
description: "Every spogo subcommand and flag, organized by topic. The machine-readable spec is at spec.md."
---

# Command Reference

Hand-written index of every `spogo` subcommand. The fully normative spec lives in [spec.md](spec.md); this page is the readable browse.

For deeper guides, see [Auth](auth.md), [Playback](playback.md), [Library](library.md), [Queue](queue.md), [Devices](devices.md), [Engines](engines.md), and [Output](output.md).

## Global flags

Apply to every command.

| Flag | Default | Purpose |
| --- | --- | --- |
| `-h`, `--help` | — | Show contextual help. |
| `--version` | — | Print the spogo version. |
| `--config <path>` | platform default | Path to a config file. |
| `--profile <name>` | `default` | Named profile (separate cookies + config). |
| `--timeout <dur>` | `10s` | HTTP timeout for any single request. |
| `--market <cc>` | account market or `US` | Two-letter market code. |
| `--language <tag>` | `en` | Language/locale. |
| `--device <name|id>` | active | Target a specific Connect device. |
| `--engine <name>` | `connect` | `auto` / `connect` / `web` / `applescript`. |
| `--json` | off | JSON output. |
| `--plain` | off | Plain (TSV) output. |
| `--no-color` | auto | Disable color in human output. |
| `-q`, `--quiet` | off | Suppress non-essential stderr. |
| `-v`, `--verbose` | off | Verbose stderr. |
| `-d`, `--debug` | off | Debug stderr (HTTP traces). |
| `--no-input` | auto when not a TTY | Refuse interactive prompts. |

Env overrides: every global flag has a `SPOGO_<NAME>` env equivalent. Two extras:

| Env | Purpose |
| --- | --- |
| `SPOGO_TOTP_SECRET_URL` | Override TOTP secret source (`http(s)` or `file://`). |
| `SPOGO_CONNECT_VERSION` | Override Connect client version sent to playback endpoints. |

## completion

Print shell completion scripts. This command does not read Spotify auth or profile configuration.

| Command | Purpose |
| --- | --- |
| `spogo completion bash` | Print Bash shell initialization. |
| `spogo completion zsh` | Print Zsh shell initialization. |
| `spogo completion fish` | Print fish shell completions. |

Load completions for the current session:

```bash
# Bash
source <(spogo completion bash)

# Zsh
source <(spogo completion zsh)

# fish
spogo completion fish | source
```

To enable them permanently, add the appropriate command to `~/.bashrc`, `~/.zshrc`, or
`~/.config/fish/config.fish`.

## auth

Cookie management. See [Auth](auth.md).

| Command | Purpose |
| --- | --- |
| `spogo auth status` | Show stored cookie state for the current profile. |
| `spogo auth import [--browser <name>] [--browser-profile <name>] [--cookie-path <file>] [--domain <host>]` | Pull cookies from a browser store. |
| `spogo auth paste [--cookie-path <file>] [--domain <suffix>] [--path <path>]` | Read cookies from stdin (interactive prompts unless `--no-input`). |
| `spogo auth clear` | Delete stored cookies for the current profile. |

## search

Browse the catalog. Each subcommand takes a query plus `--limit N` and `--offset N`.

| Command | Returns |
| --- | --- |
| `spogo search track <query>` | Tracks. |
| `spogo search album <query>` | Albums. |
| `spogo search artist <query>` | Artists. |
| `spogo search playlist <query>` | Playlists. |
| `spogo search show <query>` | Podcast shows. |
| `spogo search episode <query>` | Podcast episodes. |

Web API search and Connect's Web API fallback read Spotify's plural result containers (`tracks`, `albums`, `artists`, `playlists`, `shows`, `episodes`); CLI search types stay singular.

## info

Fetch a single item by ID, URI, or URL.

| Command | Returns |
| --- | --- |
| `spogo track info <id|url>` | One track. |
| `spogo album info <id|url>` | One album with track listing. |
| `spogo artist info <id|url>` | One artist + top tracks. |
| `spogo playlist info <id|url>` | One playlist's metadata. |
| `spogo show info <id|url>` | One show with episodes. |
| `spogo episode info <id|url>` | One episode. |

## playback

Drive what's playing. See [Playback](playback.md).

| Command | Purpose |
| --- | --- |
| `spogo play [<id|url>] [--type <kind>] [--shuffle]` | Resume, or start a track / album / playlist / show / artist. |
| `spogo pause` | Pause current playback. |
| `spogo next` | Skip to the next item. |
| `spogo prev` | Previous (restart current if past ~3s). |
| `spogo seek <ms|mm:ss>` | Seek within the current item. |
| `spogo volume <0-100>` | Set device volume. |
| `spogo shuffle <on|off>` | Toggle shuffle. |
| `spogo repeat <off|track|context>` | Set repeat mode. |
| `spogo status` | Print currently playing item, album, and device. |

## queue

Up-next list. See [Queue](queue.md).

| Command | Purpose |
| --- | --- |
| `spogo queue add <id|url>` | Append one item to the queue. |
| `spogo queue show` | Print currently playing + queued items. |
| `spogo queue clear` | Not supported by Spotify's API; use `spogo play <something>` to replace the context. |

## library

Saved tracks, albums, followed artists, owned/followed playlists. See [Library](library.md).

| Command | Purpose |
| --- | --- |
| `spogo library tracks list [--limit N]` | List saved tracks. |
| `spogo library tracks add <id|url...>` | Save tracks. |
| `spogo library tracks remove <id|url...>` | Unsave tracks. |
| `spogo library albums list [--limit N]` | List saved albums. |
| `spogo library albums add <id|url...>` | Save albums. |
| `spogo library albums remove <id|url...>` | Unsave albums. |
| `spogo library artists list [--limit N] [--after <artist-id>]` | List followed artists. |
| `spogo library artists follow <id|url...>` | Follow artists. |
| `spogo library artists unfollow <id|url...>` | Unfollow artists. |
| `spogo library playlists list [--limit N]` | List owned/followed playlists. |

## user

Read-only listening data from Spotify's web endpoints.

| Command | Purpose |
| --- | --- |
| `spogo user top-tracks [--period long_term|medium_term|short_term] [--limit N] [--offset N]` | Show Spotify top tracks by affinity ranking. |
| `spogo user history [--period long_term|medium_term|short_term] [--limit N] [--after <ms>] [--before <ms>]` | Show recently played tracks available from Spotify. |

- Top tracks are Spotify affinity rankings, not play counts.
- `long_term` = years of listening data; `medium_term` = about 6 months; `short_term` = about 4 weeks.
- Recently played is not a full archive. spogo paginates backward until it reaches `--limit` (max 200), the selected period, or Spotify's retained history. `medium_term`, `short_term`, and `--after` are local lower-bound filters.

## playlist

Mutate playlists. See [Library](library.md).

| Command | Purpose |
| --- | --- |
| `spogo playlist create <name> [--public] [--collab]` | Create a new playlist. |
| `spogo playlist add <playlist> <track...>` | Append tracks. |
| `spogo playlist remove <playlist> <track...>` | Remove tracks. |
| `spogo playlist tracks <playlist> [--limit N]` | List a playlist's items. |

`<playlist>` accepts a playlist ID, URI, URL, or owned-playlist name.

## device

Connect devices. See [Devices](devices.md).

| Command | Purpose |
| --- | --- |
| `spogo device list` | List Connect-visible devices. |
| `spogo device set <name|id>` | Transfer playback to a device. |

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Generic failure |
| `2` | Invalid usage / validation |
| `3` | Auth / cookies missing or invalid |
| `4` | Network / timeouts |

See [Output](output.md) for the full output contract.
