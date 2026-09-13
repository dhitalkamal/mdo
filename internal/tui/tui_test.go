package tui

import "testing"

func TestStripANSI(t *testing.T) {
	if got := stripANSI("\x1b[1mhi\x1b[0m"); got != "hi" {
		t.Errorf("stripANSI = %q, want %q", got, "hi")
	}
}

func TestFindMatches(t *testing.T) {
	lines := []string{"alpha", "beta", "Alpha again", "gamma"}
	got := findMatches(lines, "alpha")
	if len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Errorf("findMatches = %v, want [0 2]", got)
	}
	if findMatches(lines, "") != nil {
		t.Errorf("empty query should match nothing")
	}
}

func TestTOC(t *testing.T) {
	lines := []string{"Intro", "text", "Setup", "more", "Usage"}
	got := TOC(lines, []string{"Intro", "Setup", "Usage"})
	if len(got) != 3 {
		t.Fatalf("got %d headings, want 3", len(got))
	}
	if got[0].Line != 0 || got[1].Line != 2 || got[2].Line != 4 {
		t.Errorf("TOC lines = %+v", got)
	}
}
