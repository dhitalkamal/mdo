package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dhitalkamal/mdo/internal/img"
	"github.com/dhitalkamal/mdo/internal/term"
)

func TestRunRenders(t *testing.T) {
	var out, errb bytes.Buffer
	in := strings.NewReader("# Title\n\nhello world\n")
	if err := run([]string{"-"}, in, &out, &errb); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "Title") || !strings.Contains(out.String(), "hello world") {
		t.Errorf("render output missing content: %q", out.String())
	}
}

func TestRunListsBlocks(t *testing.T) {
	var out, errb bytes.Buffer
	in := strings.NewReader("```go\nfmt.Println(1)\n```\n")
	if err := run([]string{"-l", "-"}, in, &out, &errb); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "go") || !strings.Contains(out.String(), "fmt.Println(1)") {
		t.Errorf("list output = %q", out.String())
	}
}

func TestRunCopiesBlockAsOSC52(t *testing.T) {
	var out, errb bytes.Buffer
	in := strings.NewReader("```sh\nls -la\n```\n")
	if err := run([]string{"-c", "1", "-"}, in, &out, &errb); err != nil {
		t.Fatalf("run: %v", err)
	}
	// trailing newline is stripped so the last line waits for Enter on paste.
	want := "\x1b]52;c;bHMgLWxh\x1b\\" // base64("ls -la") = bHMgLWxh
	if out.String() != want {
		t.Errorf("copy output = %q, want %q", out.String(), want)
	}
}

func TestRunHelpExitsClean(t *testing.T) {
	var out, errb bytes.Buffer
	if err := run([]string{"--help"}, strings.NewReader(""), &out, &errb); err != nil {
		t.Fatalf("--help should return nil (clean exit), got %v", err)
	}
	help := errb.String()
	if !strings.Contains(help, "usage: mdo") {
		t.Errorf("help missing usage line: %q", help)
	}
	if !strings.Contains(help, "mcp") {
		t.Errorf("help should mention the mcp subcommand: %q", help)
	}
	if !strings.Contains(help, "example") {
		t.Errorf("help should include examples: %q", help)
	}
}

func TestRunVersion(t *testing.T) {
	var out, errb bytes.Buffer
	if err := run([]string{"--version"}, strings.NewReader(""), &out, &errb); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), version) {
		t.Errorf("version output = %q, want to contain %q", out.String(), version)
	}
}

func TestRunCopyOutOfRange(t *testing.T) {
	var out, errb bytes.Buffer
	in := strings.NewReader("no blocks here\n")
	if err := run([]string{"-c", "1", "-"}, in, &out, &errb); err == nil {
		t.Error("expected error for out-of-range copy")
	}
}

// tinyPNG makes a 2x2 png so image rendering has real bytes to decode.
func tinyPNG(t *testing.T) []byte {
	t.Helper()
	m := image.NewRGBA(image.Rect(0, 0, 2, 2))
	m.Set(0, 0, color.RGBA{255, 0, 0, 255})
	m.Set(1, 0, color.RGBA{0, 255, 0, 255})
	m.Set(0, 1, color.RGBA{0, 0, 255, 255})
	m.Set(1, 1, color.RGBA{255, 255, 0, 255})
	var b bytes.Buffer
	if err := png.Encode(&b, m); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// the pager uses half-blocks (scrollable colored text), never kitty graphics,
// which cannot scroll inside the viewport. half-blocks need truecolor.
func TestTUIImgKind(t *testing.T) {
	if got := tuiImgKind(term.LevelTrueColor); got != img.KindHalfBlock {
		t.Errorf("truecolor: got %v, want KindHalfBlock", got)
	}
	for _, lvl := range []term.Level{term.Level256, term.LevelBasic, term.LevelNone} {
		if got := tuiImgKind(lvl); got != img.KindNone {
			t.Errorf("level %v: got %v, want KindNone", lvl, got)
		}
	}
}

func TestTUIRenderDrawsImageAsHalfBlock(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pic.png"), tinyPNG(t), 0o644); err != nil {
		t.Fatal(err)
	}
	out := tuiRender([]byte("![pic](pic.png)\n"), term.LevelTrueColor, 40, dir)
	if strings.Contains(out, "[image:") {
		t.Errorf("expected a drawn image, still got a placeholder: %q", out)
	}
	if !strings.Contains(out, string(rune(0x2580))) {
		t.Errorf("expected half-block glyph, got %q", out)
	}
}

func TestTUIRenderKeepsPlaceholderWithoutTruecolor(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pic.png"), tinyPNG(t), 0o644); err != nil {
		t.Fatal(err)
	}
	out := tuiRender([]byte("![pic](pic.png)\n"), term.LevelBasic, 40, dir)
	if !strings.Contains(out, "[image:") {
		t.Errorf("non-truecolor pager should keep the text placeholder, got %q", out)
	}
}

func TestTUIRenderResolvesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "assets")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "pic.png"), tinyPNG(t), 0o644); err != nil {
		t.Fatal(err)
	}
	// base dir is dir, image path is relative to it - must resolve, not placeholder.
	out := tuiRender([]byte("![pic](assets/pic.png)\n"), term.LevelTrueColor, 40, dir)
	if strings.Contains(out, "[image:") {
		t.Errorf("relative path under BaseDir should resolve, got placeholder: %q", out)
	}
}
