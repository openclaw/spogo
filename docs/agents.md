---
title: Agents & Automation
description: "Use spogo from shell scripts, CI, cron, and AI coding agents — patterns, exit codes, and safety."
---

# Agents & Automation

spogo is built to be driven by something other than a human at a terminal — shell scripts, cron jobs, CI pipelines, AI coding agents. The same properties that make it pleasant interactively (stable JSON, plain mode, predictable exit codes, stderr-only logs) also make it safe to script.

## The contract

A script-friendly CLI must guarantee:

- **Stable output.** spogo's `--json` and `--plain` modes don't change keys/columns between releases without a major bump.
- **Stable exit codes.** `0` success, `1` generic, `2` usage, `3` auth, `4` network. Branch on these.
- **stdout is data.** stderr carries everything else. Pipes always work.
- **No interactive surprises.** When stdin is not a TTY, spogo refuses to prompt. Pass `--no-input` to be explicit.
- **Fewer public-API requests.** Connect uses internal endpoints for supported reads and playback; library mutations, playlist creation, and other Web-API-only paths can still return `429` with a retry hint.

See [Output](output.md) for the full contract.

## Wiring spogo into a shell script

```bash
#!/usr/bin/env bash
set -euo pipefail

# Make sure the configured cookie store exists
if ! spogo auth status >/dev/null 2>&1; then
  echo "spogo: cookies missing or stale; re-run 'spogo auth import'" >&2
  exit 3
fi

# For an OAuth-only Web API profile, inspect local OAuth state instead:
# spogo --engine web --auth oauth auth oauth status --json

# Capture the currently playing track ID
track_id=$(spogo status --json | jq -r '.item.id // empty')
if [[ -z "$track_id" ]]; then
  echo "Nothing playing" >&2
  exit 0
fi

# Save it
spogo library tracks add "$track_id"
```

Pin to a specific spogo version in CI to avoid silent JSON drift across releases.

## Common patterns

### Save the currently playing track

```bash
id=$(spogo status --json | jq -r '.item.id')
spogo library tracks add "$id"
```

### Build a playlist from a search

```bash
spogo playlist create "Weekly Lo-Fi"
spogo search track "lo-fi 2026" --limit 30 --plain |
  awk '{print $1}' |
  xargs spogo playlist add "Weekly Lo-Fi"
```

### Snapshot library to JSON

```bash
spogo library tracks list --limit 1000 --json > snapshots/tracks.$(date +%F).json
```

### Move playback to a specific room when leaving home

```bash
spogo device set "Phone"
```

### "Sleep timer" — pause after N minutes

```bash
sleep "${1:-1800}" && spogo pause
```

## Cron / launchd / systemd

spogo writes nothing to stdout that isn't useful and nothing to stderr unless something happened — perfect for cron tail logs.

```cron
# Snapshot liked tracks daily at 04:00
0 4 * * * /usr/local/bin/spogo library tracks list --limit 1000 --json > "$HOME/snapshots/tracks-$(date +\%F).json" 2>&1
```

For headless servers / CI runners, either copy a working cookie jar (from a machine where you ran `auth import`) into the runner's spogo config directory, or provision an OAuth token cache created by `auth oauth login` for `--engine web --auth oauth`. Both files are credentials. Do not print them or commit them.

## CI

GitHub Actions example:

```yaml
- name: Install spogo
  run: brew install steipete/tap/spogo

- name: Restore cookies
  run: |
    mkdir -p "$HOME/.config/spogo/default"
    echo "$SPOGO_COOKIES" > "$HOME/.config/spogo/default/cookies.json"
  env:
    SPOGO_COOKIES: ${{ secrets.SPOGO_COOKIES }}

- name: Snapshot library
  run: spogo library tracks list --json --limit 1000 > tracks.json
```

Treat the cookie jar like a credential — it's tied to your Spotify session.

## Coding agents

spogo is a good fit for AI coding agents (Claude Code, Codex, Cursor) because:

- **Self-documenting.** `spogo --help` and `spogo <subcommand> --help` describe the entire surface. The [Spec](spec.md) is short and stable.
- **Deterministic.** Stable JSON keys mean the agent's parsing doesn't drift across releases.
- **Safe-ish.** The destructive surface is small (`library tracks remove`, `playlist remove`, `auth clear`, `auth oauth clear`). Wrap those behind explicit confirmation in your agent prompt.

Recommended agent rules:

- Always pass `--json` or `--plain` for output the agent will parse.
- Always pass `--no-input` so spogo cannot block on a prompt.
- Branch on exit code, not stderr text.
- Pin spogo version in the agent's environment.

A starter system prompt fragment for an agent:

> You can use the `spogo` CLI to control Spotify. Always pass `--json` and `--no-input`. Read `spogo --help` and `spogo <cmd> --help` before invoking unfamiliar commands. Treat exit code `3` as "needs auth". Surface that to the user; do not launch an interactive cookie import or OAuth login automatically.

## Safety

spogo has no built-in command allowlist or read-only mode. If you're handing it to an unattended process or untrusted agent and want to restrict it:

- Run inside a separate spogo profile (`SPOGO_PROFILE=automation`) with cookies for an account that has limited permissions.
- Wrap spogo in a thin shell script that whitelists subcommands.
- Use the `applescript` engine on macOS — no cookies, no remote mutations possible (only local app control).

## Debugging an automation

Always re-run with `-v` (or `-d` for full HTTP traces) when something misbehaves — diagnostic output goes to stderr and won't pollute pipelines:

```bash
spogo -d status 2>spogo.log | jq .
```

See [Troubleshooting](troubleshooting.md).
