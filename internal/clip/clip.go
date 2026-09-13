// Package clip writes text to the system clipboard using OSC 52 terminal escape
// sequences. Because the clipboard is set by the terminal (not a local helper
// like xclip or wl-copy), it works over SSH, inside tmux, and on Termux.
package clip

import (
	"encoding/base64"
	"io"
)

// Encode returns the OSC 52 escape sequence that sets the system clipboard to s.
// Format: ESC ] 52 ; c ; <base64(s)> ST, where ST is ESC \.
func Encode(s string) string {
	b64 := base64.StdEncoding.EncodeToString([]byte(s))
	return "\x1b]52;c;" + b64 + "\x1b\\"
}

// Copy writes the OSC 52 sequence for s to w (typically the terminal / stdout).
func Copy(w io.Writer, s string) error {
	_, err := io.WriteString(w, Encode(s))
	return err
}
