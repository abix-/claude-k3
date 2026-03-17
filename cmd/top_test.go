package cmd

import (
	"reflect"
	"testing"
)

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"a\nb\nc", []string{"a", "b", "c"}},
		{"a\n\nb", []string{"a", "b"}},
		{"", nil},
		{"single", []string{"single"}},
		{"trailing\n", []string{"trailing"}},
	}
	for _, tt := range tests {
		got := splitLines(tt.input)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitLines(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
