package clip

import (
	"bytes"
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"ascii", "hello", "\x1b]52;c;aGVsbG8=\x1b\\"},
		{"empty", "", "\x1b]52;c;\x1b\\"},
		{"utf8", "café", "\x1b]52;c;Y2Fmw6k=\x1b\\"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Encode(tt.in); got != tt.want {
				t.Errorf("Encode(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCopyWritesEncoded(t *testing.T) {
	var buf bytes.Buffer
	if err := Copy(&buf, "hello"); err != nil {
		t.Fatalf("Copy returned error: %v", err)
	}
	if got, want := buf.String(), Encode("hello"); got != want {
		t.Errorf("Copy wrote %q, want %q", got, want)
	}
}
