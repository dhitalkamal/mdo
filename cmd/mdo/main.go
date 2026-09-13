// Command mdo renders a Markdown file in the terminal, and can copy its fenced
// code blocks to the system clipboard over OSC 52 (works over SSH / Termux).
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhitalkamal/mdo/internal/blocks"
	"github.com/dhitalkamal/mdo/internal/clip"
	"github.com/dhitalkamal/mdo/internal/img"
	"github.com/dhitalkamal/mdo/internal/mcp"
	"github.com/dhitalkamal/mdo/internal/render"
	"github.com/dhitalkamal/mdo/internal/runner"
	"github.com/dhitalkamal/mdo/internal/term"
	"github.com/dhitalkamal/mdo/internal/tui"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "mdo:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// `mdo mcp` starts the Model Context Protocol server on stdio for AI agents.
	if len(args) > 0 && args[0] == "mcp" {
		return mcp.Serve(stdin, stdout, version)
	}

	fs := flag.NewFlagSet("mdo", flag.ContinueOnError)
	fs.SetOutput(stderr)
	copyN := fs.Int("copy", 0, "copy the Nth fenced code block to the clipboard (OSC 52) and exit")
	list := fs.Bool("list", false, "list fenced code blocks and exit")
	runIt := fs.Bool("run", false, "step through fenced shell blocks, confirming each before it runs")
	tuiMode := fs.Bool("tui", false, "open an interactive pager (scroll, search, contents)")
	noColor := fs.Bool("no-color", false, "disable color output")
	width := fs.Int("width", 0, "wrap width in columns (0 = auto-detect)")
	showVersion := fs.Bool("version", false, "print version and exit")
	fs.IntVar(copyN, "c", 0, "shorthand for --copy")
	fs.BoolVar(list, "l", false, "shorthand for --list")
	fs.BoolVar(runIt, "r", false, "shorthand for --run")
	fs.BoolVar(tuiMode, "t", false, "shorthand for --tui")
	fs.Usage = func() {
		fmt.Fprint(stderr, `usage: mdo [flags] [file]
       mdo mcp

Render a markdown file (or stdin) in the terminal.

examples:
  mdo README.md            render a file
  mdo --tui README.md      interactive pager (scroll, / search, t contents)
  mdo --list README.md     list fenced code blocks
  mdo --copy 1 README.md   copy a code block to the clipboard (OSC 52)
  mdo --run script.md      step through shell blocks, confirming each
  mdo mcp                  run the MCP server on stdio (for AI agents)

flags:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil // usage already printed; help is a clean exit
		}
		return err
	}

	if *showVersion {
		fmt.Fprintf(stdout, "mdo %s\n", version)
		return nil
	}

	if (*runIt || *tuiMode) && (fs.Arg(0) == "" || fs.Arg(0) == "-") {
		return fmt.Errorf("--run and --tui need a file argument (stdin is used for interaction)")
	}

	src, err := readInput(fs.Arg(0), stdin)
	if err != nil {
		return err
	}

	switch {
	case *list:
		return listBlocks(stdout, src)
	case *copyN > 0:
		return copyBlock(stdout, stderr, src, *copyN)
	case *runIt:
		return runBlocks(stdin, stdout, src)
	case *tuiMode:
		return runTUI(src, colorLevel(*noColor, stdout), fdOf(stdout))
	default:
		level := colorLevel(*noColor, stdout)
		w := *width
		if w == 0 {
			w = term.Width(fdOf(stdout), 80)
		}
		base := "."
		if f := fs.Arg(0); f != "" && f != "-" {
			base = filepath.Dir(f)
		}
		opts := render.Options{Level: level, Width: w, Images: imgKind(level, stdout), BaseDir: base}
		_, err := io.WriteString(stdout, render.Render(src, opts))
		return err
	}
}

func readInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "" || path == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(path)
}

func listBlocks(stdout io.Writer, src []byte) error {
	for _, b := range blocks.Extract(src) {
		lang := b.Lang
		if lang == "" {
			lang = "text"
		}
		if _, err := fmt.Fprintf(stdout, "%d\t%s\t%s\n", b.Index, lang, firstLine(b.Code)); err != nil {
			return err
		}
	}
	return nil
}

func copyBlock(stdout, stderr io.Writer, src []byte, n int) error {
	bs := blocks.Extract(src)
	if n > len(bs) {
		return fmt.Errorf("no code block %d (found %d)", n, len(bs))
	}
	// strip trailing newlines so pasting into a shell leaves the last command
	// at the prompt for you to review and run, instead of auto-executing it.
	code := strings.TrimRight(bs[n-1].Code, "\n")
	if err := clip.Copy(stdout, code); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "copied code block %d to clipboard\n", n)
	return nil
}

// runBlocks steps through the shell code blocks, confirming each before it runs.
func runBlocks(stdin io.Reader, stdout io.Writer, src []byte) error {
	var shell []blocks.Block
	for _, b := range blocks.Extract(src) {
		if runner.IsShell(b.Lang) {
			shell = append(shell, b)
		}
	}
	if len(shell) == 0 {
		_, err := fmt.Fprintln(stdout, "no runnable shell code blocks found")
		return err
	}
	return runner.Run(shell, stdin, stdout, runner.ShellExecutor)
}

// runTUI opens the interactive pager on the rendered document.
func runTUI(src []byte, level term.Level, fd int) error {
	if level == term.LevelNone {
		level = term.LevelBasic // the pager is interactive; keep some color
	}
	content := render.Render(src, render.Options{Level: level, Width: term.Width(fd, 80)})
	return tui.Run(content, headingTexts(src))
}

// headingTexts pulls ATX heading text (lines starting with #) for the TOC.
func headingTexts(src []byte) []string {
	var hs []string
	for _, ln := range strings.Split(string(src), "\n") {
		t := strings.TrimLeft(ln, " ")
		if strings.HasPrefix(t, "#") {
			if h := strings.TrimSpace(strings.TrimLeft(t, "#")); h != "" {
				hs = append(hs, h)
			}
		}
	}
	return hs
}

// colorLevel decides color, disabling it when output is piped or NO_COLOR is set.
func colorLevel(noColor bool, out io.Writer) term.Level {
	if noColor {
		return term.LevelNone
	}
	f, ok := out.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return term.LevelNone
	}
	return term.ColorLevel(os.Getenv)
}

// imgKind reports the terminal's image capability, off when piped or NO_COLOR.
func imgKind(level term.Level, out io.Writer) img.Kind {
	if level == term.LevelNone {
		return img.KindNone
	}
	f, ok := out.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return img.KindNone
	}
	return img.Capable(os.Getenv, level == term.LevelTrueColor)
}

func fdOf(out io.Writer) int {
	if f, ok := out.(*os.File); ok {
		return int(f.Fd())
	}
	return -1
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
