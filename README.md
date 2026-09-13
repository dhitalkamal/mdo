# mdo

markdown, do.

Render a Markdown file in the terminal and step through, vet, and run its command
blocks. Built for people who live in the terminal on any device (SSH, Termux, a
bare TTY), and designed so AI agents can drive it too: an agent proposes a
runbook, you vet and run each step.

Status: work in progress. v1 scope is render + OSC 52 clipboard copy.

## Why

Existing tools either render Markdown or run its code blocks, but none are built
for the terminal-everywhere, human-vets-the-agent workflow:

- copies code blocks to your real clipboard even over SSH / Termux (OSC 52)
- works anywhere a terminal runs, single static binary, no runtime deps
- plain-text and git-native: your docs stay yours, no lock-in

## Build

```sh
make build
./mdo README.md
```

## Usage

```sh
mdo README.md            # render a markdown file
mdo --list README.md     # list the fenced code blocks
mdo --copy 1 README.md   # copy the first code block to the clipboard (OSC 52)
```

## Status

Early development. Not yet released.
