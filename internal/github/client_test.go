package github

import (
	"testing"

	gh "github.com/google/go-github/v68/github"
)

func label(name string) *gh.Label {
	return &gh.Label{Name: &name}
}

func TestParseIssueLabels(t *testing.T) {
	tests := []struct {
		name      string
		labels    []*gh.Label
		wantState string
		wantOwner string
	}{
		{"owner only = owner is state", []*gh.Label{label("claude-a")}, "claude-a", "claude-a"},
		{"ready no owner", []*gh.Label{label("ready")}, "ready", ""},
		{"needs-review no owner", []*gh.Label{label("needs-review")}, "needs-review", ""},
		{"needs-human with owner", []*gh.Label{label("needs-human"), label("claude-b")}, "needs-human", "claude-b"},
		{"codex owner", []*gh.Label{label("codex-1")}, "codex-1", "codex-1"},
		{"ready with owner", []*gh.Label{label("ready"), label("claude-c")}, "ready", "claude-c"},
		{"no workflow labels", []*gh.Label{label("bug"), label("enhancement")}, "", ""},
		{"owner: prefix not detected", []*gh.Label{label("owner:claude-a")}, "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotState, gotOwner := parseIssueLabels(tc.labels)
			if gotState != tc.wantState {
				t.Errorf("state: got %q, want %q", gotState, tc.wantState)
			}
			if gotOwner != tc.wantOwner {
				t.Errorf("owner: got %q, want %q", gotOwner, tc.wantOwner)
			}
		})
	}
}

func TestGetOwnedIssuesFiltersNeedsHuman(t *testing.T) {
	parked := []*gh.Label{label("needs-human"), label("claude-a")}
	state, owner := parseIssueLabels(parked)
	if owner == "" {
		t.Fatal("needs-human issue has no owner parsed")
	}
	if owner != "" && state != "needs-human" {
		t.Errorf("needs-human issue would be picked up as orphan")
	}
}

func TestIsK3sAgent(t *testing.T) {
	tests := []struct {
		owner string
		want  bool
	}{
		{"claude-a", true},
		{"claude-z", true},
		{"claude-1", false},
		{"claude-10", false},
		{"codex-1", false},
		{"", false},
	}
	for _, tc := range tests {
		if got := isK3sAgent(tc.owner); got != tc.want {
			t.Errorf("isK3sAgent(%q) = %v, want %v", tc.owner, got, tc.want)
		}
	}
}
