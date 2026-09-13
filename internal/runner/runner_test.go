package runner

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/dhitalkamal/mdo/internal/blocks"
)

func mkblocks(codes ...string) []blocks.Block {
	var bs []blocks.Block
	for i, c := range codes {
		bs = append(bs, blocks.Block{Index: i + 1, Lang: "sh", Code: c})
	}
	return bs
}

func TestRunExecutesOnYes(t *testing.T) {
	var ran []string
	exec := func(code string, out io.Writer) error { ran = append(ran, code); return nil }
	var out bytes.Buffer
	if err := Run(mkblocks("echo hi\n"), strings.NewReader("y\n"), &out, exec); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 1 || ran[0] != "echo hi\n" {
		t.Errorf("ran = %v, want [\"echo hi\\n\"]", ran)
	}
	if !strings.Contains(out.String(), "echo hi") {
		t.Errorf("block not shown to user: %q", out.String())
	}
}

func TestRunSkips(t *testing.T) {
	ran := 0
	exec := func(code string, out io.Writer) error { ran++; return nil }
	var out bytes.Buffer
	Run(mkblocks("echo hi\n"), strings.NewReader("s\n"), &out, exec)
	if ran != 0 {
		t.Errorf("exec called %d times, want 0 on skip", ran)
	}
	if !strings.Contains(out.String(), "skipped") {
		t.Errorf("missing skip message: %q", out.String())
	}
}

func TestRunQuitStopsIteration(t *testing.T) {
	ran := 0
	exec := func(code string, out io.Writer) error { ran++; return nil }
	var out bytes.Buffer
	Run(mkblocks("a\n", "b\n"), strings.NewReader("q\n"), &out, exec)
	if ran != 0 {
		t.Errorf("exec called on immediate quit")
	}
	if strings.Count(out.String(), "block ") > 1 {
		t.Errorf("iterated past quit: %q", out.String())
	}
}

func TestRunRepromptsOnInvalid(t *testing.T) {
	ran := 0
	exec := func(code string, out io.Writer) error { ran++; return nil }
	var out bytes.Buffer
	Run(mkblocks("x\n"), strings.NewReader("huh\ny\n"), &out, exec)
	if ran != 1 {
		t.Errorf("exec called %d times, want 1 after reprompt", ran)
	}
	if !strings.Contains(out.String(), "please answer") {
		t.Errorf("missing reprompt: %q", out.String())
	}
}

func TestRunCopyEmitsOSC52(t *testing.T) {
	exec := func(code string, out io.Writer) error { return nil }
	var out bytes.Buffer
	Run(mkblocks("ls\n"), strings.NewReader("c\n"), &out, exec)
	if !strings.Contains(out.String(), "\x1b]52;c;") {
		t.Errorf("copy did not emit OSC52: %q", out.String())
	}
}

func TestRunEOFStops(t *testing.T) {
	ran := 0
	exec := func(code string, out io.Writer) error { ran++; return nil }
	var out bytes.Buffer
	Run(mkblocks("a\n"), strings.NewReader(""), &out, exec) // no input -> EOF -> stop
	if ran != 0 {
		t.Errorf("exec called on EOF")
	}
}

func TestIsShell(t *testing.T) {
	for _, l := range []string{"", "sh", "bash", "zsh", "console"} {
		if !IsShell(l) {
			t.Errorf("IsShell(%q) = false, want true", l)
		}
	}
	for _, l := range []string{"go", "python", "json"} {
		if IsShell(l) {
			t.Errorf("IsShell(%q) = true, want false", l)
		}
	}
}
