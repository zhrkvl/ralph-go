package agent

import (
	"slices"
	"testing"
)

func TestBuildArgs(t *testing.T) {
	base := []string{
		"--dangerously-skip-permissions",
		"--print",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
	}
	tests := []struct {
		name   string
		model  string
		effort string
		want   []string
	}{
		{"no flags", "", "", base},
		{"model only", "opus", "", slices.Concat(base, []string{"--model", "opus"})},
		{"effort only", "", "high", slices.Concat(base, []string{"--effort", "high"})},
		{"both", "sonnet", "max", slices.Concat(base, []string{"--model", "sonnet", "--effort", "max"})},
		{"effort is not validated", "", "bogus", slices.Concat(base, []string{"--effort", "bogus"})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildArgs(tt.model, tt.effort); !slices.Equal(got, tt.want) {
				t.Errorf("buildArgs(%q, %q) = %v, want %v", tt.model, tt.effort, got, tt.want)
			}
		})
	}
}
