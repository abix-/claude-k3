package operator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/abix-/k3sc/internal/config"
	"github.com/abix-/k3sc/internal/dispatch"
	"github.com/abix-/k3sc/internal/github"
	"github.com/abix-/k3sc/internal/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const MaxFailures = 3

// Verbose enables full structured logging from controller-runtime.
// Set via --verbose flag on the operator command.
var Verbose bool

var edt = time.FixedZone("EDT", -4*3600)

// olog prints a concise timestamped log line: "18:27:22 [prefix] message"
func olog(prefix, format string, args ...any) {
	t := time.Now().In(edt).Format("15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s [%s] %s\n", t, prefix, msg)
}

func Scanner(ctx context.Context, c client.Client, namespace string) {
	logger := log.FromContext(ctx).WithName("scanner")
	minInterval := config.C.Scan.MinInterval.Duration
	maxInterval := config.C.Scan.MaxInterval.Duration
	interval := minInterval
	logger.Info("starting github scanner", "interval", interval)

	hadWork := scan(ctx, c, namespace)
	if !hadWork {
		interval = nextBackoff(interval, maxInterval)
	}

	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			hadWork = scan(ctx, c, namespace)
			if hadWork {
				interval = minInterval
			} else {
				interval = nextBackoff(interval, maxInterval)
			}
			olog("scanner","next scan in %s", interval)
			timer.Reset(interval)
		}
	}
}

func nextBackoff(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		next = max
	}
	return next
}

func scan(ctx context.Context, c client.Client, namespace string) bool {
	eligible, err := github.GetEligibleIssues(ctx)
	if err != nil {
		olog("scanner","github error: %v", err)
		return false
	}

	var existing AgentJobList
	if err := c.List(ctx, &existing, client.InNamespace(namespace)); err != nil {
		olog("scanner","list tasks error: %v", err)
		return false
	}

	// build state from existing tasks
	activeIssues := map[string]bool{}
	failCounts := map[string]int{}
	usedSlots := []int{}
	for _, t := range existing.Items {
		key := fmt.Sprintf("%s-%d", t.Spec.RepoName, t.Spec.IssueNumber)
		if !IsTerminal(t.Status.Phase) && t.Status.Phase != "" {
			activeIssues[key] = true
			usedSlots = append(usedSlots, t.Spec.Slot)
		}
		if t.Status.Phase == TaskPhaseFailed {
			failCounts[key]++
		}
	}

	maxSlots := dispatch.MaxSlots()

	// create tasks one at a time, updating usedSlots after each
	for _, issue := range eligible {
		key := fmt.Sprintf("%s-%d", issue.Repo.Name, issue.Number)
		if activeIssues[key] {
			continue
		}
		if failCounts[key] >= MaxFailures {
			olog("scanner","%s blocked after %d failures", key, failCounts[key])
			continue
		}

		slot := dispatch.FindFreeSlotFromList(usedSlots, maxSlots)
		if slot == -1 {
			olog("scanner","no free slots")
			break
		}

		agent := types.AgentName(slot)
		epoch := time.Now().Unix()
		name := fmt.Sprintf("%s-%d-%d", strings.ReplaceAll(issue.Repo.Name, "/", "-"), issue.Number, epoch)

		task := &AgentJob{
			TypeMeta: metav1.TypeMeta{
				APIVersion: GroupVersion.String(),
				Kind:       "AgentJob",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: AgentJobSpec{
				Repo:        fmt.Sprintf("%s/%s", issue.Repo.Owner, issue.Repo.Name),
				RepoName:    issue.Repo.Name,
				IssueNumber: issue.Number,
				RepoURL:     issue.Repo.CloneURL(),
				Slot:        slot,
				Agent:       agent,
				OriginState: issue.State,
			},
			Status: AgentJobStatus{
				Phase: TaskPhasePending,
			},
		}

		if err := c.Create(ctx, task); err != nil {
			olog("scanner","create %s: %v", name, err)
			continue
		}
		olog("scanner","created %s (slot %d, %s)", name, slot, agent)

		// update in-memory state so next iteration sees this slot as used
		usedSlots = append(usedSlots, slot)
		activeIssues[key] = true
	}

	// TTL cleanup
	for i := range existing.Items {
		t := &existing.Items[i]
		if !IsTerminal(t.Status.Phase) {
			continue
		}
		if time.Since(t.CreationTimestamp.Time) > config.C.Scan.TaskTTL.Duration {
			if err := c.Delete(ctx, t); err != nil {
				olog("scanner","cleanup %s: %v", t.Name, err)
			} else {
				olog("scanner","cleaned up %s", t.Name)
			}
		}
	}

	return len(eligible) > 0
}
