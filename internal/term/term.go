// Package term detects terminal capabilities: color support level and width.
// Detection is env-driven and pure where possible so it is easy to test.
package term

import (
	"strings"

	xterm "golang.org/x/term"
)

// Level is how much color the terminal supports.
type Level int

const (
	LevelNone      Level = iota // no color (NO_COLOR, or a dumb terminal)
	LevelBasic                  // 16 ANSI colors
	Level256                    // 256 colors
	LevelTrueColor              // 24-bit color
)

// ColorLevel decides color support from environment variables. env is an
// accessor (pass os.Getenv in real use) so callers can test it with a fake.
func ColorLevel(env func(string) string) Level {
	if env("NO_COLOR") != "" {
		return LevelNone
	}
	term := env("TERM")
	if term == "dumb" {
		return LevelNone
	}
	switch strings.ToLower(env("COLORTERM")) {
	case "truecolor", "24bit":
		return LevelTrueColor
	}
	if strings.Contains(term, "256color") {
		return Level256
	}
	return LevelBasic
}

// Width returns the terminal width for file descriptor fd, or fallback when the
// size cannot be determined (e.g. output is piped to a file).
func Width(fd, fallback int) int {
	if w, _, err := xterm.GetSize(fd); err == nil && w > 0 {
		return w
	}
	return fallback
}

// IsTerminal reports whether fd is attached to a terminal.
func IsTerminal(fd int) bool { return xterm.IsTerminal(fd) }
