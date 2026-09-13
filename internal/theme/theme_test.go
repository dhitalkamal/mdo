package theme

import (
	"strings"
	"testing"

	"github.com/dhitalkamal/mdo/internal/term"
)

func TestNoneLevelIsPlain(t *testing.T) {
	p := For(term.LevelNone)
	if got := p.Bold.Apply("x"); got != "x" {
		t.Errorf("None Bold.Apply = %q, want plain %q", got, "x")
	}
	if got := p.H1.Apply("Title"); got != "Title" {
		t.Errorf("None H1.Apply = %q, want plain text", got)
	}
}

func TestColoredWrapsAndResets(t *testing.T) {
	p := For(term.LevelBasic)
	got := p.Bold.Apply("x")
	if !strings.HasPrefix(got, "\x1b[") || !strings.HasSuffix(got, reset) || !strings.Contains(got, "x") {
		t.Errorf("Basic Bold.Apply = %q, want escape-wrapped 'x' ending in reset", got)
	}
}
