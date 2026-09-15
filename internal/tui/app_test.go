package tui

import "testing"

func TestIsPromiseLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"bare marker", promiseMarker, true},
		{"stamped marker", "21:28:15 " + promiseMarker, true},
		{"padded marker", "  " + promiseMarker + "  ", true},
		{"styled marker", "\033[1m" + promiseMarker + "\033[0m", true},
		{
			"prose declining to emit the marker",
			"21:28:15 Not all stories are done, so this iteration ends normally " +
				"without the " + promiseMarker + " marker.",
			false,
		},
		{"prose quoting the marker", "reply with " + promiseMarker, false},
		{"unrelated line", "21:28:15 [result] success | 81 turns", false},
		{"empty line", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPromiseLine(tt.line); got != tt.want {
				t.Errorf("isPromiseLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}
