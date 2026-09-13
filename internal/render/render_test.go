package render

import (
	"strings"
	"testing"

	"github.com/dhitalkamal/mdo/internal/term"
)

func plain(src string) string {
	return Render([]byte(src), Options{Level: term.LevelNone, Width: 80})
}

func TestHeadingAndParagraph(t *testing.T) {
	out := plain("# Hello\n\nSome text here.\n")
	if !strings.Contains(out, "Hello") || !strings.Contains(out, "Some text here.") {
		t.Errorf("missing content: %q", out)
	}
	if strings.Contains(out, "\x1b") {
		t.Errorf("LevelNone must be plain, got escapes: %q", out)
	}
	if strings.Contains(out, "#") {
		t.Errorf("heading marker not stripped: %q", out)
	}
}

func TestEmphasisMarkersStripped(t *testing.T) {
	out := plain("This is **bold** and *italic*.\n")
	if !strings.Contains(out, "bold") || !strings.Contains(out, "italic") {
		t.Errorf("emphasis text missing: %q", out)
	}
	if strings.Contains(out, "**") || strings.Contains(out, "*italic*") {
		t.Errorf("markers not stripped: %q", out)
	}
}

func TestColoredHasEscapes(t *testing.T) {
	out := Render([]byte("**bold**\n"), Options{Level: term.LevelBasic, Width: 80})
	if !strings.Contains(out, "\x1b[1m") {
		t.Errorf("expected bold escape, got %q", out)
	}
}

func TestCodeBlockKeepsCode(t *testing.T) {
	out := plain("```go\nfmt.Println(1)\n```\n")
	if !strings.Contains(out, "fmt.Println(1)") {
		t.Errorf("code content missing: %q", out)
	}
}

func TestListItems(t *testing.T) {
	out := plain("- one\n- two\n- three\n")
	for _, w := range []string{"one", "two", "three"} {
		if !strings.Contains(out, w) {
			t.Errorf("list item %q missing: %q", w, out)
		}
	}
}

func TestBlockquoteGutter(t *testing.T) {
	out := plain("> quoted line\n")
	if !strings.Contains(out, "quoted line") || !strings.Contains(out, "|") {
		t.Errorf("blockquote gutter/content missing: %q", out)
	}
}

func TestParagraphWraps(t *testing.T) {
	out := Render([]byte("aaaa bbbb cccc dddd\n"), Options{Level: term.LevelNone, Width: 9})
	if !strings.Contains(out, "\n") {
		t.Errorf("expected wrapping at width 9, got one line: %q", out)
	}
}
