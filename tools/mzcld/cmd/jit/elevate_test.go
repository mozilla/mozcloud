package jit

import (
	"testing"
	"time"
)

func TestFormatRemaining(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{-1 * time.Second, "expired"},
		{0, "00:00"},
		{30 * time.Second, "00:30"},
		{5*time.Minute + 30*time.Second, "05:30"},
		{59*time.Minute + 59*time.Second, "59:59"},
		{1 * time.Hour, "1:00:00"},
		{2*time.Hour + 15*time.Minute, "2:15:00"},
		{8*time.Hour + 0*time.Minute + 1*time.Second, "8:00:01"},
	}
	for _, tt := range tests {
		got := formatRemaining(tt.d)
		if got != tt.want {
			t.Errorf("formatRemaining(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
