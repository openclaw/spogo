---
title: Install
description: "Install spogo via Homebrew, go install, or a release binary. macOS, Linux, and Windows are all supported."
---

# Install

spogo ships as a single static Go binary. Pick whichever path matches how you usually install CLIs.

## Homebrew (macOS, Linux)

```bash
brew install steipete/tap/spogo
```

That's it — the formula pulls a signed binary from the latest GitHub release.

## go install (any platform)

```bash
go install github.com/steipete/spogo/cmd/spogo@latest
```

Builds from source against your local Go toolchain. Requires Go 1.25.12+.

## Pre-built release binaries

Grab a tarball or zip for your OS/arch from the [releases page](https://github.com/openclaw/spogo/releases) and drop the `spogo` binary somewhere on `PATH`:

```bash
curl -L https://github.com/openclaw/spogo/releases/latest/download/spogo_$(uname -s)_$(uname -m).tar.gz | tar xz
sudo mv spogo /usr/local/bin/
spogo --version
```

## Build from source

```bash
git clone https://github.com/openclaw/spogo.git
cd spogo
make spogo
./spogo --version
```

## Verify

```bash
spogo --version
spogo --help
```

If `--help` lists `auth`, `search`, `play`, `library`, `playlist`, `device`, and friends, you're done. Next stop: [Quickstart](quickstart.md).

## Uninstall

- Homebrew: `brew uninstall spogo`
- go install: `rm $(which spogo)`
- Manual: delete the binary; remove `~/Library/Application Support/spogo` (macOS), `~/.config/spogo` (Linux), or `%APPDATA%\spogo` (Windows) to clear cached cookies and config.
