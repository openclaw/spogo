# Vision

spogo is a dependable, script-first Spotify power CLI. It should make search,
playback, queue, library, playlist, and listening-data workflows composable from
a terminal without requiring a developer application or a hosted service.

## Priorities

1. **Reliable automation.** Non-interactive commands, deterministic output, and
   useful exit codes matter more than decorative interactive behavior.
2. **Resilient access.** Prefer Spotify Connect and web-player surfaces where
   they avoid public API limits; keep explicit engine selection and bounded
   fallbacks when those private surfaces drift.
3. **Local control.** Credentials, profiles, caches, and history stay on the
   user's machine. spogo does not become a hosted credential relay or account
   service.
4. **Cross-platform behavior.** macOS, Linux, and Windows share one CLI contract.
   Platform-specific engines may add capabilities without weakening that common
   surface.
5. **Small, testable scope.** Favor focused commands that compose well over a
   daemon, desktop UI, media downloader, or speculative abstraction layer.

## Compatibility contract

- `--json` keys, `--plain` fields, stdout/stderr separation, exit codes, command
  names, flags, config, and environment variables are public interfaces.
- Compatible releases may add fields or commands. Renaming or removing existing
  machine-readable output requires a major release and migration notes.
- Explicit engine behavior must remain predictable. Automatic fallbacks may
  recover from rate limits or unsupported operations, but must not turn a
  read-only command into a mutation or hide an authentication failure.
- Internal Spotify endpoints are implementation details, not excuses for silent
  breakage. Changes need focused fixtures plus authenticated live proof when the
  affected service boundary is available.

## Security and privacy

- Reuse user-owned browser sessions; never print cookie or token values, accept
  secrets as ordinary command-line arguments, or send credentials to a spogo
  service.
- Keep profiles isolated. Debug output may explain requests and fallbacks but
  must redact credentials and private session material.
- Do not bypass account entitlements, DRM, or media-delivery restrictions.

## Change policy

Bug fixes, compatibility repairs, performance work, and bounded new commands fit
when they preserve the contracts above and have an end-to-end verification path.
New default behavior, engines, persistent data formats, credential flows, or
security/privacy policy require an explicit product decision. User-visible
changes need docs, changelog coverage, regression tests where practical, and
proof against the built CLI before release.
