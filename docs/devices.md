---
title: Devices
description: "List Spotify Connect devices, switch active playback, target a specific device per command."
---

# Devices

Every Spotify-capable thing you've signed in on — phone, desktop app, web player, smart speaker, console — is a Spotify Connect device. spogo can list them and route playback to whichever you want.

## device list

```bash
spogo device list
spogo device list --plain
spogo device list --json
```

Prints every device Spotify Connect currently knows about, with the active one marked. Plain mode is one device per line: `id`, `name`, `is_active`. JSON also includes `type` and `volume_percent`.

## device set

```bash
spogo device set "Kitchen"
spogo device set 0d1841b0976bae2a3a310dd74c0f3df354899bc8
```

Transfers playback to the named device (case-insensitive exact match) or device ID. If the current Connect state has no origin device, spogo falls back to the Web API transfer endpoint instead of failing.

## --device flag (per-command)

`--device` is a global flag. The Web API sends its value as a device ID on mutations; Connect uses it to find a target for `play` when no device is active:

```bash
spogo play spotify:track:... --device "Kitchen"
spogo volume 30 --device "MacBook Pro"
```

To consistently select a device for subsequent controls, use `device set` first. `status` reports the current playback state and does not query an arbitrary device selected by this flag.

## Default device

Set a per-shell default with the env var:

```bash
export SPOGO_DEVICE="Kitchen"
spogo play spotify:track:...           # goes to Kitchen
spogo play spotify:track:... --device "Phone"   # overrides
```

## Discovery tips

- A device only shows up after it has been opened/played to recently. If your speaker isn't listed, open Spotify on it once.
- The Spotify desktop app shows up as the device name from the OS (`MacBook Pro`, `peter-laptop`).
- Web players appear as `Web Player (Chrome)` and similar; they don't persist after the tab closes.
- Sonos / Google Cast / AirPlay endpoints appear when they're being used by a Spotify session — not always at idle.

## Errors

- **`device not found`** — open the device's Spotify session once, then re-run.
- **`PREMIUM_REQUIRED`** — Connect transfer needs Premium.
- **`Connect state has no origin device`** — happens when no device is currently active; spogo retries via the Web API transfer.
