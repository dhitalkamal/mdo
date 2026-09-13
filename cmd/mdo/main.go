// Command mdo renders a Markdown file in the terminal, and can copy its fenced
// code blocks to the system clipboard over OSC 52 (works over SSH / Termux).
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dhitalkamal/mdo/internal/blocks"
	"github.com/dhitalkamal/mdo/internal/clip"
	"github.com/dhitalkamal/mdo/internal/render"
	"github.com/dhitalkamal/mdo/internal/term"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "mdo:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("mdo", flag.ContinueOnError)
	fs.SetOutput(stderr)
	copyN := fs.Int("copy", 0, "copy the Nth fenced code block to the clipboard (OSC 52) and exit")
	list := fs.Bool("list", false, "list fenced code blocks and exit")
	noColor := fs.Bool("no-color", false, "disable color output")
	width := fs.Int("width", 0, "wrap width in columns (0 = auto-detect)")
	fs.IntVar(copyN, "c", 0, "shorthand for --copy")
	fs.BoolVar(list, "l", false, "shorthand for --list")
	fs.Usage = func() {
		fmt.Fprint(stderr, "usage: mdo [flags] [file]\n\n"+
			"render a markdown file (or stdin) in the terminal.\n\nflags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
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
	default:
		level := colorLevel(*noColor, stdout)
		w := *width
		if w == 0 {
			w = term.Width(fdOf(stdout), 80)
		}
		_, err := io.WriteString(stdout, render.Render(src, render.Options{Level: level, Width: w}))
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
