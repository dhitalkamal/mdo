package img

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

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

func TestCapable(t *testing.T) {
	kitty := func(k string) string {
		if k == "KITTY_WINDOW_ID" {
			return "1"
		}
		return ""
	}
	if Capable(kitty, false) != KindKitty {
		t.Error("want KindKitty when KITTY_WINDOW_ID set")
	}
	empty := func(string) string { return "" }
	if Capable(empty, true) != KindHalfBlock {
		t.Error("want KindHalfBlock on truecolor")
	}
	if Capable(empty, false) != KindNone {
		t.Error("want KindNone otherwise")
	}
}

func TestRenderHalfBlock(t *testing.T) {
	out, err := Render(tinyPNG(t), KindHalfBlock, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\x1b[38;2;") || !strings.Contains(out, string(rune(0x2580))) {
		t.Errorf("half-block output missing truecolor/block glyph: %q", out)
	}
}

func TestRenderKitty(t *testing.T) {
	out, err := Render(tinyPNG(t), KindKitty, 40)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "\x1b_G") {
		t.Errorf("kitty output should start with graphics escape, got %q", out[:min(24, len(out))])
	}
}

func TestRenderNone(t *testing.T) {
	if _, err := Render(tinyPNG(t), KindNone, 40); err == nil {
		t.Error("KindNone should return an error")
	}
}
