package format

import (
	"strings"
	"testing"
	"time"

	"github.com/abix-/k3sc/internal/types"
)

func TestFmtTime_nil(t *testing.T) {
	if got := FmtTime(nil); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestFmtTime_nonNil(t *testing.T) {
	ts := time.Date(2025, 1, 1, 15, 4, 0, 0, time.UTC)
	got := FmtTime(&ts)
	if got == "" {
		t.Fatal("expected non-empty string")
	}
}

func TestFmtDuration_nilStart(t *testing.T) {
	if got := FmtDuration(nil, nil); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestFmtDuration_withEnd(t *testing.T) {
	start := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 1, 12, 2, 30, 0, time.UTC)
	got := FmtDuration(&start, &end)
	if got != "2m 30s" {
		t.Fatalf("expected %q, got %q", "2m 30s", got)
	}
}

func TestCountPhases(t *testing.T) {
	pods := []types.AgentPod{
		{Phase: types.PhaseRunning},
		{Phase: types.PhasePending},
		{Phase: types.PhaseSucceeded},
		{Phase: types.PhaseFailed},
		{Phase: types.PhaseUnknown},
	}
	running, completed, failed := CountPhases(pods)
	if running != 2 || completed != 1 || failed != 1 {
		t.Fatalf("expected 2/1/1, got %d/%d/%d", running, completed, failed)
	}
}

func TestIssueLink(t *testing.T) {
	repo := types.Repo{Owner: "abix-", Name: "k3s-claude"}
	link := IssueLink(repo, 5)
	if !strings.Contains(link, "abix-/k3s-claude/issues/5") {
		t.Fatalf("expected link to contain repo path, got %q", link)
	}
	if !strings.Contains(link, "#5") {
		t.Fatalf("expected link to contain #5, got %q", link)
	}
}

func TestPRLink(t *testing.T) {
	repo := types.Repo{Owner: "abix-", Name: "k3s-claude"}
	link := PRLink(repo, 3)
	if !strings.Contains(link, "abix-/k3s-claude/pull/3") {
		t.Fatalf("expected link to contain pull path, got %q", link)
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("hello", 10); got != "hello" {
		t.Fatalf("expected %q, got %q", "hello", got)
	}
	if got := Truncate("hello world", 8); got != "hello..." {
		t.Fatalf("expected %q, got %q", "hello...", got)
	}
}
