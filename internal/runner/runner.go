// Package runner steps through a document's fenced shell blocks interactively,
// showing each and asking before it runs - the "human vets the agent" workflow.
package runner

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/dhitalkamal/mdo/internal/blocks"
	"github.com/dhitalkamal/mdo/internal/clip"
)

// Executor runs a block's code, streaming output to out.
type Executor func(code string, out io.Writer) error

type action int

const (
	actionRun action = iota
	actionCopy
	actionSkip
	actionQuit
	actionUnknown
)

// Run steps through bs, prompting on in and writing to out; exec runs a block
// when the user confirms. Quitting or exhausted input stops the walk.
func Run(bs []blocks.Block, in io.Reader, out io.Writer, exec Executor) error {
	r := bufio.NewReader(in)
	for _, b := range bs {
		showBlock(out, b)
		switch prompt(r, out) {
		case actionRun:
			fmt.Fprintln(out, "  running...")
			if err := exec(b.Code, out); err != nil {
				fmt.Fprintf(out, "  exited with error: %v\n", err)
			}
		case actionCopy:
			_ = clip.Copy(out, strings.TrimRight(b.Code, "\n"))
			fmt.Fprintln(out, "  copied to clipboard")
		case actionSkip:
			fmt.Fprintln(out, "  skipped")
		case actionQuit:
			return nil
		}
	}
	return nil
}

func showBlock(out io.Writer, b blocks.Block) {
	lang := b.Lang
	if lang == "" {
		lang = "sh"
	}
	fmt.Fprintf(out, "\nblock %d (%s):\n", b.Index, lang)
	for _, line := range strings.Split(strings.TrimRight(b.Code, "\n"), "\n") {
		fmt.Fprintf(out, "    %s\n", line)
	}
}

// prompt asks until it gets a valid answer; exhausted input is treated as quit.
func prompt(r *bufio.Reader, out io.Writer) action {
	for {
		fmt.Fprint(out, "run? [y]es / [c]opy / [s]kip / [q]uit: ")
		line, err := r.ReadString('\n')
		if a := parseAction(line); a != actionUnknown {
			return a
		}
		if err != nil {
			return actionQuit
		}
		fmt.Fprintln(out, "  please answer y, c, s, or q")
	}
}

func parseAction(line string) action {
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes", "r", "run":
		return actionRun
	case "c", "copy":
		return actionCopy
	case "s", "skip", "n", "no":
		return actionSkip
	case "q", "quit":
		return actionQuit
	default:
		return actionUnknown
	}
}

// ShellExecutor runs code with the user's shell ($SHELL, or /bin/sh).
func ShellExecutor(code string, out io.Writer) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell, "-c", code)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// IsShell reports whether a fenced block's language runs as a shell command.
func IsShell(lang string) bool {
	switch strings.ToLower(lang) {
	case "", "sh", "bash", "zsh", "shell", "console", "shell-session":
		return true
	default:
		return false
	}
}
