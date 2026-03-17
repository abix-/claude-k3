package k8s

import (
	"testing"
	"time"

	"github.com/abix-/k3sc/internal/types"
)

func TestFindRecentUsageLimitPodFromLogsPrefersMostRecentFailure(t *testing.T) {
	now := time.Date(2026, time.March, 17, 12, 0, 0, 0, time.UTC)
	older := now.Add(-10 * time.Minute)
	recent := now.Add(-2 * time.Minute)

	pods := []types.AgentPod{
		{Name: "old", Issue: 8, Repo: types.Repo{Name: "endless"}, Phase: types.PhaseFailed, Finished: &older},
		{Name: "new", Issue: 11, Repo: types.Repo{Name: "k3sc"}, Phase: types.PhaseFailed, Finished: &recent},
	}
	logs := map[string]string{
		"old": UsageLimitMessage,
		"new": UsageLimitMessage,
	}

	pod := FindRecentUsageLimitPodFromLogs(now, 15*time.Minute, pods, logs)
	if pod == nil || pod.Name != "new" {
		t.Fatalf("FindRecentUsageLimitPodFromLogs() = %#v, want pod %q", pod, "new")
	}
}

func TestFindRecentUsageLimitPodFromLogsIgnoresOldAndHealthyPods(t *testing.T) {
	now := time.Date(2026, time.March, 17, 12, 0, 0, 0, time.UTC)
	old := now.Add(-20 * time.Minute)
	recent := now.Add(-3 * time.Minute)

	pods := []types.AgentPod{
		{Name: "old-failed", Issue: 8, Repo: types.Repo{Name: "k3sc"}, Phase: types.PhaseFailed, Finished: &old},
		{Name: "recent-success", Issue: 10, Repo: types.Repo{Name: "k3sc"}, Phase: types.PhaseSucceeded, Finished: &recent},
		{Name: "recent-failed", Issue: 11, Repo: types.Repo{Name: "k3sc"}, Phase: types.PhaseFailed, Finished: &recent},
	}
	logs := map[string]string{
		"old-failed":     UsageLimitMessage,
		"recent-success": UsageLimitMessage,
		"recent-failed":  "some other error",
	}

	pod := FindRecentUsageLimitPodFromLogs(now, 15*time.Minute, pods, logs)
	if pod != nil {
		t.Fatalf("FindRecentUsageLimitPodFromLogs() = %#v, want nil", pod)
	}
}

func TestParseUsageLimitResetTimeSameDay(t *testing.T) {
	now := time.Date(2026, time.March, 17, 16, 6, 0, 0, time.UTC)
	resetAt, ok := ParseUsageLimitResetTime(now, "You're out of extra usage · resets 5pm (UTC)")
	if !ok {
		t.Fatal("ParseUsageLimitResetTime() = not ok, want ok")
	}
	want := time.Date(2026, time.March, 17, 17, 0, 0, 0, time.UTC)
	if !resetAt.Equal(want) {
		t.Fatalf("ParseUsageLimitResetTime() = %s, want %s", resetAt, want)
	}
}

func TestFilterUnreportedFinishedPodsSkipsReported(t *testing.T) {
	finished := time.Now()
	pods := []types.AgentPod{
		{Name: "pod-a", Issue: 10, Phase: types.PhaseSucceeded, Finished: &finished},
		{Name: "pod-b", Issue: 11, Phase: types.PhaseFailed, Finished: &finished},
		{Name: "pod-c", Issue: 12, Phase: types.PhaseSucceeded, Finished: &finished},
	}
	reported := map[string]bool{"pod-a": true}

	result := FilterUnreportedFinishedPods(pods, reported)
	if len(result) != 2 {
		t.Fatalf("FilterUnreportedFinishedPods() = %d pods, want 2", len(result))
	}
	for _, p := range result {
		if p.Name == "pod-a" {
			t.Fatal("FilterUnreportedFinishedPods() returned already-reported pod pod-a")
		}
	}
}

func TestFilterUnreportedFinishedPodsSkipsRunning(t *testing.T) {
	finished := time.Now()
	pods := []types.AgentPod{
		{Name: "pod-run", Issue: 10, Phase: types.PhaseRunning},
		{Name: "pod-done", Issue: 11, Phase: types.PhaseSucceeded, Finished: &finished},
	}

	result := FilterUnreportedFinishedPods(pods, map[string]bool{})
	if len(result) != 1 || result[0].Name != "pod-done" {
		t.Fatalf("FilterUnreportedFinishedPods() = %v, want [pod-done]", result)
	}
}

func TestFilterUnreportedFinishedPodsSkipsZeroIssue(t *testing.T) {
	finished := time.Now()
	pods := []types.AgentPod{
		{Name: "pod-no-issue", Issue: 0, Phase: types.PhaseSucceeded, Finished: &finished},
		{Name: "pod-has-issue", Issue: 5, Phase: types.PhaseFailed, Finished: &finished},
	}

	result := FilterUnreportedFinishedPods(pods, map[string]bool{})
	if len(result) != 1 || result[0].Name != "pod-has-issue" {
		t.Fatalf("FilterUnreportedFinishedPods() = %v, want [pod-has-issue]", result)
	}
}

func TestParseUsageLimitResetTimeRollsToNextDay(t *testing.T) {
	now := time.Date(2026, time.March, 17, 18, 0, 0, 0, time.UTC)
	resetAt, ok := ParseUsageLimitResetTime(now, "You're out of extra usage · resets 5pm (UTC)")
	if !ok {
		t.Fatal("ParseUsageLimitResetTime() = not ok, want ok")
	}
	want := time.Date(2026, time.March, 18, 17, 0, 0, 0, time.UTC)
	if !resetAt.Equal(want) {
		t.Fatalf("ParseUsageLimitResetTime() = %s, want %s", resetAt, want)
	}
}
