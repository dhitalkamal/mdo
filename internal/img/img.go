// Package img renders images in the terminal: the kitty graphics protocol when
// available, otherwise unicode half-blocks (works on any truecolor terminal),
// so images degrade gracefully instead of failing on SSH/Termux/TTY.
package img

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Kind is the best image method the terminal supports.
type Kind int

const (
	KindNone      Kind = iota // no image support (text placeholder)
	KindHalfBlock             // unicode half-blocks (truecolor)
	KindKitty                 // kitty graphics protocol
)

// upperHalfBlock is U+2580, built from its code point so the source stays ASCII.
var upperHalfBlock = string(rune(0x2580))

// Capable picks the best method from the environment. truecolor enables the
// half-block fallback when kitty is not detected.
func Capable(env func(string) string, truecolor bool) Kind {
	if env("KITTY_WINDOW_ID") != "" || strings.Contains(env("TERM"), "kitty") {
		return KindKitty
	}
	if truecolor {
		return KindHalfBlock
	}
	return KindNone
}

// Load reads image bytes from a local path or an http(s) URL.
func Load(src string) ([]byte, error) {
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(src)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("http %d fetching %s", resp.StatusCode, src)
		}
		return io.ReadAll(io.LimitReader(resp.Body, 20<<20)) // 20 MB cap
	}
	return os.ReadFile(src)
}

// Render turns image bytes into terminal output using the given method. maxCols
// bounds the display width in terminal columns.
func Render(data []byte, kind Kind, maxCols int) (string, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	switch kind {
	case KindKitty:
		return kitty(src), nil
	case KindHalfBlock:
		return halfBlock(src, maxCols), nil
	default:
		return "", fmt.Errorf("no image support")
	}
}

// resize does a simple nearest-neighbour downscale to w x h.
func resize(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	b := src.Bounds()
	for y := 0; y < h; y++ {
		sy := b.Min.Y + y*b.Dy()/h
		for x := 0; x < w; x++ {
			sx := b.Min.X + x*b.Dx()/w
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

// halfBlock renders using the upper-half-block glyph, packing two vertical
// pixels per cell via foreground (top) and background (bottom) truecolor.
func halfBlock(src image.Image, maxCols int) string {
	b := src.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw < 1 || ih < 1 {
		return ""
	}
	cols := maxCols
	if iw < cols {
		cols = iw
	}
	if cols < 1 {
		cols = 1
	}
	rows := cols * ih / iw / 2
	if rows < 1 {
		rows = 1
	}
	rz := resize(src, cols, rows*2)
	var sb strings.Builder
	for ry := 0; ry < rows; ry++ {
		for cx := 0; cx < cols; cx++ {
			tr, tg, tb, _ := rz.At(cx, ry*2).RGBA()
			br, bg, bb, _ := rz.At(cx, ry*2+1).RGBA()
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm%s",
				tr>>8, tg>>8, tb>>8, br>>8, bg>>8, bb>>8, upperHalfBlock)
		}
		sb.WriteString("\x1b[0m\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// kitty encodes the image as PNG and emits the kitty graphics escape (chunked).
func kitty(src image.Image) string {
	if src.Bounds().Dx() > 1000 { // keep the escape sane
		h := 1000 * src.Bounds().Dy() / src.Bounds().Dx()
		src = resize(src, 1000, h)
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, src)
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	var sb strings.Builder
	const chunk = 4096
	for i := 0; i < len(b64); i += chunk {
		end := i + chunk
		if end > len(b64) {
			end = len(b64)
		}
		more := 0
		if end < len(b64) {
			more = 1
		}
		if i == 0 {
			fmt.Fprintf(&sb, "\x1b_Gf=100,a=T,m=%d;%s\x1b\\", more, b64[i:end])
		} else {
			fmt.Fprintf(&sb, "\x1b_Gm=%d;%s\x1b\\", more, b64[i:end])
		}
	}
	return sb.String()
}
