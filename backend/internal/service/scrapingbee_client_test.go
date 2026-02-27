package service

import (
	"strings"
	"testing"
)

func TestIsCFChallengePage(t *testing.T) {
	cfMarker := "challenge-platform/scripts/jsd/main.js"

	tests := []struct {
		name     string
		html     string
		expected bool
	}{
		{
			name:     "pure CF challenge page (small + marker)",
			html:     strings.Repeat("x", 10_000) + cfMarker,
			expected: true,
		},
		{
			name:     "real page with CF bot-management script injected (large)",
			html:     strings.Repeat("x", 250_000) + cfMarker,
			expected: false,
		},
		{
			name:     "normal page without CF marker",
			html:     strings.Repeat("x", 800_000),
			expected: false,
		},
		{
			name:     "empty page",
			html:     "",
			expected: false,
		},
		{
			name:     "small page without CF marker",
			html:     "<html><body>Not found</body></html>",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCFChallengePage(tt.html)
			if got != tt.expected {
				t.Errorf("isCFChallengePage() = %v, want %v (html size=%d)", got, tt.expected, len(tt.html))
			}
		})
	}
}
