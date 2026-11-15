<div align="center">

[![CI](https://img.shields.io/github/actions/workflow/status/jadegopher/trabbit/ci.yml?branch=main&style=flat-square&label=tests)](https://github.com/jadegopher/trabbit/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/jadegopher/trabbit?style=flat-square&logo=go)](go.mod)
[![Release](https://img.shields.io/github/v/release/jadegopher/trabbit?style=flat-square)](https://github.com/jadegopher/trabbit/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)

[Install](#install) · [Connect](#connect) · [Contribute](CONTRIBUTING.md)

</div>

![tRabbit demo: create a queue, edit and publish JSON, then explore queue statistics](docs/demo.gif)

A TUI for administering RabbitMQ, built as an alternative to the management web
UI. Inspect queues, watch traffic and publish messages without leaving your
terminal.

**Early days:** `trabbit` currently covers the **Queues** tab. The other management
tabs are placeholders. I plan to add more as people show interest or I need more
functionality myself. Have something in mind? [Open an issue](https://github.com/jadegopher/trabbit/issues).

## What you can do

- **Manage queues** — create, delete and purge; filter by name, regex or virtual host.
- **Watch live activity** — message counts, throughput charts, consumers, bindings
  and runtime statistics, with configurable refresh and chart settings.
- **Publish comfortably** — a full-screen payload editor with JSON beautification,
  text selection, undo/redo, and editable message properties and headers.
- **Use keys or a mouse** — click to focus and select, scroll panels, and drag to
  select text in the editor. Contextual shortcuts keep actions close at hand.

## Install

### macOS — Homebrew

With [Homebrew](https://brew.sh/) installed:

```sh
brew tap jadegopher/trabbit https://github.com/jadegopher/trabbit
brew install --cask jadegopher/trabbit/trabbit
```

Update with `brew upgrade --cask jadegopher/trabbit/trabbit`.

### Windows — Scoop

With [Scoop](https://scoop.sh/) installed, run in PowerShell:

```powershell
scoop bucket add trabbit https://github.com/jadegopher/trabbit
scoop install trabbit/trabbit
```

Update with `scoop update trabbit`. Both package managers select the matching
amd64 or arm64 binary.

### Linux — download with curl

For Linux amd64, download into the directory where you want to keep the binary:

```sh
curl -fL https://github.com/jadegopher/trabbit/releases/download/v0.1.0/trabbit_0.1.0_linux_amd64.tar.gz | tar -xz trabbit
sudo ln -sf "$PWD/trabbit" /usr/local/bin/trabbit
```

Use `arm64` instead of `amd64` for ARM machines. On macOS, the same commands work
with `darwin` instead of `linux`. Choose your version from
[Releases](https://github.com/jadegopher/trabbit/releases); rerun the download to
upgrade. Keep the extracted binary in place—the symlink points to it.

On Windows without Scoop, download the matching `windows_amd64.zip` or
`windows_arm64.zip` from Releases, extract it, and add its directory to your user
`PATH`.

### Go or source — all platforms

With [Go](https://go.dev/dl/) installed:

```sh
go install github.com/jadegopher/trabbit/cmd/trabbit@latest
```

To build from source, follow [CONTRIBUTING.md](CONTRIBUTING.md#development).
Archives for every supported OS and architecture are also available on
[GitHub Releases](https://github.com/jadegopher/trabbit/releases).

## Connect

Start `trabbit` for the login form, or connect directly:

```sh
trabbit --url 'http://localhost:15672' --user guest --pass guest
```

Use the RabbitMQ **management HTTP(S) endpoint**, usually port `15672`. The
[management plugin](https://www.rabbitmq.com/docs/management) must be enabled.
See [CLI usage](docs/cli.md) for connection options, or run `trabbit --help`.

On the Queues tab, `a` adds a queue, `f` filters, and `p` opens the publisher.
Use `Tab` to switch panels and `[` / `]` to cycle detail tabs. In the payload
editor, `Ctrl+B` beautifies JSON, `Ctrl+G` opens editing help, and `F5` publishes.
Press `Ctrl+C` twice in a row to quit.

---

Built with [Go](https://go.dev/) and [Bubble Tea](https://github.com/charmbracelet/bubbletea).
Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).
Licensed under [MIT](LICENSE).
