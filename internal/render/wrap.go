package render

import "strings"

// visibleLen counts runes in s, ignoring ANSI escape sequences so styled text
// measures by what the user actually sees.
func visibleLen(s string) int {
	n, inEsc := 0, false
	for _, r := range s {
		if inEsc {
			// an SGR sequence ends at its final letter (e.g. the 'm' in \x1b[1m)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		if r == '\x1b' {
			inEsc = true
			continue
		}
		n++
	}
	return n
}

// wrap word-wraps s to width visible columns. Words split on whitespace; escape
// sequences do not count toward the width. width <= 0 disables wrapping.
func wrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	var b strings.Builder
	lineLen := 0
	for i, w := range words {
		wl := visibleLen(w)
		switch {
		case i == 0:
			b.WriteString(w)
			lineLen = wl
		case lineLen+1+wl > width:
			b.WriteByte('\n')
			b.WriteString(w)
			lineLen = wl
		default:
			b.WriteByte(' ')
			b.WriteString(w)
			lineLen += 1 + wl
		}
	}
	return b.String()
}
