package tui

import "testing"

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		current, total int
		width          int
		want           string
	}{
		{0, 10, 10, "[          ]"},
		{5, 10, 10, "[=====>    ]"},
		{10, 10, 10, "[==========]"},
		{3, 10, 10, "[==>       ]"},
	}

	for _, tt := range tests {
		got := renderProgressBar(tt.current, tt.total, tt.width)
		if got != tt.want {
			t.Errorf("renderProgressBar(%d, %d, %d) = %q, want %q",
				tt.current, tt.total, tt.width, got, tt.want)
		}
	}
}
