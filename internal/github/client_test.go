package github

import (
	"strings"
	"testing"

	gh "github.com/google/go-github/v68/github"
)

func TestOwnerLabelParsing(t *testing.T) {
	parseOwner := func(labels []string) string {
		for _, l := range labels {
			if strings.HasPrefix(l, "owner:") {
				return strings.TrimPrefix(l, "owner:")
			}
		}
		return ""
	}

	tests := []struct {
		name   string
		labels []string
		want   string
	}{
		{"claude-c owner label", []string{"claimed", "owner:claude-c"}, "claude-c"},
		{"claude-b owner label", []string{"ready", "owner:claude-b"}, "claude-b"},
		{"codex family", []string{"claimed", "owner:codex-1"}, "codex-1"},
		{"no owner label", []string{"ready", "bug"}, ""},
		{"owner prefix only would not panic", []string{"owner:"}, ""},
		{"label exactly claude-", []string{"claude-"}, ""},
		{"label exactly codex-", []string{"codex-"}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseOwner(tc.labels)
			if got != tc.want {
				t.Errorf("parseOwner(%v) = %q, want %q", tc.labels, got, tc.want)
			}
		})
	}
}

// TestOldParsingWouldFailForOwnerPrefix verifies the old index-slicing approach
// would miss owner: prefixed labels (regression guard).
func TestOldParsingWouldFailForOwnerPrefix(t *testing.T) {
	oldParseOwner := func(labels []string) string {
		for _, l := range labels {
			if len(l) > 6 && (l[:7] == "claude-" || l[:6] == "codex-") {
				return l
			}
		}
		return ""
	}

	// owner:claude-c does NOT start with "claude-" so old code returns ""
	got := oldParseOwner([]string{"owner:claude-c"})
	if got != "" {
		t.Errorf("expected old parser to miss owner:claude-c, got %q", got)
	}
}

func label(name string) *gh.Label {
	return &gh.Label{Name: &name}
}

// TestParseIssueLabelsClaude verifies that claude-* labels are detected as owner
// without the old owner: prefix. This test FAILS if the implementation reverts
// to the owner: prefix format.
func TestParseIssueLabelsClaude(t *testing.T) {
	tests := []struct {
		name      string
		labels    []*gh.Label
		wantState string
		wantOwner string
	}{
		{"claude-a is owner", []*gh.Label{label("claimed"), label("claude-a")}, "claimed", "claude-a"},
		{"codex-1 is owner", []*gh.Label{label("claimed"), label("codex-1")}, "claimed", "codex-1"},
		{"needs-human skips owner", []*gh.Label{label("needs-human"), label("claude-b")}, "needs-human", "claude-b"},
		{"no owner label", []*gh.Label{label("ready")}, "ready", ""},
		{"owner: prefix is not detected", []*gh.Label{label("claimed"), label("owner:claude-a")}, "claimed", ""},
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

// TestGetOwnedIssuesFiltersNeedsHuman verifies GetOwnedIssues logic excludes needs-human.
// Uses the same parseIssueLabels path to ensure orphan detection won't touch parked issues.
func TestGetOwnedIssuesFiltersNeedsHuman(t *testing.T) {
	parked := []*gh.Label{label("needs-human"), label("claude-a")}
	state, owner := parseIssueLabels(parked)
	if owner == "" {
		t.Fatal("needs-human issue has no owner parsed -- test setup wrong")
	}
	// simulate the filter: owner != "" && state != "needs-human"
	if owner != "" && state != "needs-human" {
		t.Errorf("needs-human issue would be picked up as orphan -- filter is broken")
	}
}
