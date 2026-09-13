# mdo

markdown, do.

Render a Markdown file in the terminal and step through, vet, and run its command
blocks. Built for people who live in the terminal on any device (SSH, Termux, a
bare TTY), and designed so AI agents can drive it too: an agent proposes a
runbook, you vet and run each step.

Status: work in progress. v1: render, OSC 52 clipboard copy, and an interactive
step-through runner that vets each shell block before running it.

## Why

Existing tools either render Markdown or run its code blocks, but none are built
for the terminal-everywhere, human-vets-the-agent workflow:

- copies code blocks to your real clipboard even over SSH / Termux (OSC 52)
- works anywhere a terminal runs, single static binary, no runtime deps
- plain-text and git-native: your docs stay yours, no lock-in

## Install

One-line install (downloads the right binary to `~/.local/bin`):

```sh
curl -fsSL https://raw.githubusercontent.com/dhitalkamal/mdo/main/install.sh | sh
```

With Go:

```sh
go install github.com/dhitalkamal/mdo/cmd/mdo@latest
```

Homebrew (macOS / Linux):

```sh
brew install dhitalkamal/tap/mdo
```

Arch Linux: `mdo-bin` on the AUR. Termux (Android), Raspberry Pi, and servers:
use the one-line installer, or grab the linux arm64 binary from the
[releases page](https://github.com/dhitalkamal/mdo/releases). Prebuilt binaries
cover linux amd64/arm64 and macOS amd64/arm64.

## Build from source

```sh
git clone https://github.com/dhitalkamal/mdo
cd mdo && make build
./mdo README.md
```

## Usage

```sh
mdo README.md            # render a markdown file
mdo --list README.md     # list the fenced code blocks
mdo --copy 1 README.md   # copy the first code block to the clipboard (OSC 52)
mdo --run README.md      # step through shell blocks, confirming each before it runs
```

With `--run`, mdo shows each shell block and asks `[y]es / [c]opy / [s]kip / [q]uit`
before it runs - so an AI agent can propose a runbook and you vet every step.

## Use as an MCP server (AI agents)

mdo speaks the Model Context Protocol over stdio, so AI agents can render markdown
and inspect a document's code blocks natively:

```sh
mdo mcp
```

Register it with an MCP client, e.g. Claude Code:

```sh
claude mcp add mdo -- mdo mcp
```

Tools exposed: `render_markdown`, `list_code_blocks`, `get_code_block` (each takes
`path` or inline `content`). This is the point of mdo: an agent proposes, and the
tool + human vet and run.

## Status

Early development.
