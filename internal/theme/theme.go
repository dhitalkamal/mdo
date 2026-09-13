// Package theme maps markdown elements to ANSI styles, chosen by the terminal's
// color level. At LevelNone every style is a no-op, so output stays plain text.
package theme

import "github.com/dhitalkamal/mdo/internal/term"

// Style wraps text in an opening and closing escape sequence.
type Style struct{ open, close string }

// Apply wraps text. A zero Style (open == "") returns text unchanged, which is
// what makes NO_COLOR / dumb terminals get clean plain text.
func (s Style) Apply(text string) string {
	if s.open == "" {
		return text
	}
	return s.open + text + s.close
}

// Palette holds one Style per rendered element.
type Palette struct {
	H1, H2, H3   Style
	Bold, Italic Style
	Strike       Style
	Code         Style // inline code
	Quote        Style
	Link         Style
	Bullet       Style
	Rule         Style
}

const reset = "\x1b[0m"

func sgr(codes string) Style { return Style{open: "\x1b[" + codes + "m", close: reset} }

// For returns the palette for a color level. Colored levels share a 16-color
// scheme (works from LevelBasic up); LevelNone returns all no-op styles.
func For(level term.Level) Palette {
	if level == term.LevelNone {
		return Palette{}
	}
	return Palette{
		H1:     sgr("1;36"), // bold cyan
		H2:     sgr("1;35"), // bold magenta
		H3:     sgr("1;34"), // bold blue
		Bold:   sgr("1"),
		Italic: sgr("3"),
		Strike: sgr("9"),    // strikethrough
		Code:   sgr("33"),   // yellow
		Quote:  sgr("2;32"), // dim green
		Link:   sgr("4;34"), // underline blue
		Bullet: sgr("36"),   // cyan
		Rule:   sgr("2"),    // dim
	}
}
