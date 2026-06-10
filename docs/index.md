---
title: Overview
permalink: /
description: "spogo is a Spotify power CLI for search, playback, library, playlists, devices, and scripting — auth via browser cookies, output as human, plain, or JSON."
---

# spogo

A single Go binary for Spotify on the command line. Search the catalog, drive playback, manage your library and playlists, pick devices, and pipe stable JSON or plain output into anything.

## Why spogo

- **No app registration.** spogo authenticates with the cookies your browser already has — `spogo auth import --browser chrome` and you're authenticated. No client ID, no redirect URI, no developer dashboard.
- **No tight rate limits.** Talks to Spotify's internal web endpoints (the same ones `open.spotify.com` uses), so search, info, and playback are usable for automation that the public Web API would throttle.
- **Predictable output.** `--json` for tools, `--plain` for `awk`/`cut`, color human output by default. `NO_COLOR`, `TERM=dumb`, and `--no-color` all respected.
- **Multiple engines.** `connect` (internal), `web` (public Web API), `auto` (connect first, fall back to web), and `applescript` (drive Spotify.app on macOS) — pick what works.
- **Built for agents.** Stable exit codes, structured errors, machine output — drop spogo into a shell script or hand it to a coding agent and it'll behave.

## Pick your path

- **Trying it.** Read [Install](install.md) then [Quickstart](quickstart.md). About five minutes from `brew install` to playing a track from the terminal.
- **Cookies are confusing.** Read [Auth](auth.md) for browser import, manual paste, the WSL fallback, and the `auth status` checks.
- **Picking an engine.** Read [Engines](engines.md) for the tradeoffs between `connect`, `web`, `auto`, and `applescript`.
- **Wiring up a script or agent.** Read [Output](output.md) for the JSON/plain contract, then [Agents](agents.md) for end-to-end automation patterns.
- **Looking up a flag.** Open the [Command Reference](commands.md) — every subcommand is listed with its flags. The full machine-readable spec lives in [Spec](spec.md).

## Status

Actively developed. The [CHANGELOG](https://github.com/openclaw/spogo/blob/main/CHANGELOG.md) tracks every shipping release. Current line is `v0.10.x`.

## Out of scope

- MCP server — this is a CLI.
- Hosted runtime, web UI, or GUI.
- A library — spogo is a tool, not a Go SDK; internal packages are not stable.

Released under the [MIT license](https://github.com/openclaw/spogo/blob/main/LICENSE). Not affiliated with Spotify. Spotify is a trademark of Spotify AB.
