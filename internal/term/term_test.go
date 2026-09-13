package term

import "testing"

func TestColorLevel(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want Level
	}{
		{"no_color wins", map[string]string{"NO_COLOR": "1", "COLORTERM": "truecolor"}, LevelNone},
		{"dumb terminal", map[string]string{"TERM": "dumb"}, LevelNone},
		{"truecolor", map[string]string{"TERM": "xterm-256color", "COLORTERM": "truecolor"}, LevelTrueColor},
		{"24bit", map[string]string{"COLORTERM": "24bit"}, LevelTrueColor},
		{"256color", map[string]string{"TERM": "screen-256color"}, Level256},
		{"basic default", map[string]string{"TERM": "xterm"}, LevelBasic},
		{"empty", map[string]string{}, LevelBasic},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := func(k string) string { return tt.env[k] }
			if got := ColorLevel(env); got != tt.want {
				t.Errorf("ColorLevel(%v) = %d, want %d", tt.env, got, tt.want)
			}
		})
	}
}

func TestWidthFallback(t *testing.T) {
	// fd -1 is never a terminal, so Width must return the fallback.
	if got := Width(-1, 80); got != 80 {
		t.Errorf("Width(-1, 80) = %d, want 80", got)
	}
}
