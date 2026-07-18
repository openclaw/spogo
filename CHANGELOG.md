# Changelog

## 0.10.3 - 2026-07-18

### Highlights

- Make release and Homebrew-installed binaries report their actual tagged version

### Fixed

- Stamp the CLI version from the GoReleaser tag while retaining a `dev` fallback for every unstamped build
- Remove stale hardcoded `0.10.1` version labels from the CLI and specification

### Release engineering

- Upgrade the unified release caller to `openclaw/release-workflows@v1.0.0-alpha.13`, including exact verified asset names and SHA-256 digests in the Homebrew handoff

## 0.10.2 - 2026-07-18

### Highlights

- Ship the first spogo release through the unified signed pipeline, using the OpenClaw Foundation Developer ID and Apple notarization for every macOS binary

### Release engineering

- Replace the tag-triggered release workflow with a thin, manually dispatched caller pinned to `openclaw/release-workflows@v1.0.0-alpha.12`
- Bind published release notes byte-for-byte to this dated changelog section and verify immutable artifacts before publication
- Hand verified release metadata to the Homebrew tap updater after publication

## 0.10.1 - 2026-07-17

### Highlights

- Add native shell completion generation for Bash, Zsh, and Fish
- Make missing browser authentication fail with the documented exit code for reliable automation
- Surface Spotify rate-limit cooldown hints so callers can back off without guessing

### CLI and authentication

- Add Bash, Zsh, and Fish shell completions and show supported shells in command help (`#32`, thanks @kk-spartans)
- Return exit code `3` when browser cookies are unavailable instead of collapsing authentication failures into a generic command error
- Surface Spotify `Retry-After` hints in API errors with bounded seconds and HTTP-date parsing (`#41`, thanks @clawSean)

### Reliability and maintenance

- Build CI and release artifacts with Go 1.25.12 so upstream standard-library security fixes are included
- Refresh direct and transitive Go dependencies, including Kong, go-toml, x/crypto, x/sys, and modernc.org/libc
- Refresh GitHub Actions and pinned dead-code, linting, and formatting tools

### Documentation

- Define product direction, compatibility guarantees, security boundaries, and change policy in `VISION.md`
- Update source-install requirements and command documentation for the refreshed CLI and Go toolchain

## 0.10.0 - 2026-06-10

- Add user top tracks and retained listening history commands (`#29`, thanks @hibachipapi)
- Fix Firefox cookie imports when browser expiry values cannot be serialized (`#31`, thanks @meoyawn)

## 0.9.0 - 2026-05-10

- Improve Connect playback latency by caching web auth, client tokens, and the active command route between invocations (`#25`, thanks @kk-spartans)
- Fix warm Connect commands from cached active routes by persisting the registered spogo sender device.
- Fix Connect device volumes so `device list` and status output report `0-100` percentages instead of Spotify's raw volume scale.

## 0.3.1 - 2026-05-10

- Fix Connect Pathfinder track metadata extraction for explicit ratings, nested durations, and playability (`#26`, thanks @theDimZone)
- Add generated `llms.txt` docs index for agent-friendly documentation discovery
- Update release automation/docs for the OpenClaw repository move

## 0.3.0 - 2026-05-05

- Add `auth paste`, wire `--no-input`, and improve cookie diagnostics/cleanup (`#5`, thanks @im-zayan)
- Add `play --shuffle`, Connect library/playlist support, and context-aware Connect play payloads (`#15`, thanks @StandardGage)
- Fix Connect track artist extraction for nested artist containers and minimal artist fragments (`#7`, thanks @joelbdavies)
- Fix silent `auth import` failures by retrying Spotify auth cookie lookup across related hosts and surfacing browser warnings (`#13`)
- Fix `device set` when Connect state has no origin device by falling back to Web API transfer (`#8`)
- Fix Connect liked-track listing via `fetchLibraryTracks` with Web API fallback on payload drift (`#16`, thanks @masonc15)
- Fix Connect play when no device is active by falling back to Web API playback (`#21`, thanks @prashanthbala)
- Fix Connect volume changes by sending the volume endpoint as `PUT` (`#24`, thanks @cavit99)
- Fix sparse status/search metadata so track artists and albums are populated consistently across engines.
- Fix Connect `--device` playback when no device is active without falling back to rate-limited Web API playback.
- Fix `auth paste --no-input` by accepting the documented flag order.
- Fix playlist add/remove 429s by using Connect playlist mutations with writable-playlist checks and fallback coverage across engines (`#20`).
- Release prep: bump CLI/spec version to `0.3.0`

## 0.2.0 - 2026-01-07

- Add `applescript` engine for direct Spotify.app control on macOS (thanks @adam91holt)
- CI: bump golangci-lint-action to support golangci-lint v2

## 0.1.0 - 2026-01-02

- Kong-powered CLI with global flags, config profiles, and env overrides
- Auth commands: cookie status/import/clear with browser/profile selection
- Cookie-based auth via steipete/sweetcookie (file cache + browser sources)
- Search tracks/albums/artists/playlists/shows/episodes
- Item info for track/album/artist/playlist/show/episode
- Playback control: play/pause/next/prev/seek/volume/shuffle/repeat/status
- Artist play (top tracks; falls back to search)
- Queue add/show
- Library list/add/remove for tracks/albums; follow/unfollow artists; playlists list
- Playlist management: create/add/remove/track list
- Device list and transfer/set
- Engines: connect (internal), web (Web API), auto (connect → web fallback)
- Rate-limit fallback on 429s where supported
- Output: human color + `--plain` + `--json` (NO_COLOR/TERM aware)
- GitHub Actions CI, linting, formatting, and coverage gate
