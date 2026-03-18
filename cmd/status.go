package cmd

import (
	"fmt"
	"sync"

	"github.com/abix-/k3sc/internal/config"
	"github.com/abix-/k3sc/internal/github"
	"github.com/abix-/k3sc/internal/k8s"
	"github.com/abix-/k3sc/internal/types"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "One-line cluster and issue summary",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cs, err := k8s.NewClient()
	if err != nil {
		return err
	}

	var (
		pods   []types.AgentPod
		issues []types.Issue
		prs    []types.PullRequest
		mu     sync.Mutex
		wg     sync.WaitGroup
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		p, _ := k8s.GetAgentPods(ctx, cs)
		mu.Lock()
		pods = p
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		i, _ := github.GetAllOpenIssues(ctx)
		mu.Lock()
		issues = i
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		p, _ := github.GetOpenPRs(ctx)
		mu.Lock()
		prs = p
		mu.Unlock()
	}()
	wg.Wait()

	running := 0
	for _, p := range pods {
		if p.Phase == types.PhaseRunning {
			running++
		}
	}

	counts := map[string]int{}
	for _, i := range issues {
		counts[i.State]++
	}

	parts := []string{fmt.Sprintf("%d running", running)}
	for _, state := range []string{"ready", "needs-review", "needs-human"} {
		if n := counts[state]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, state))
		}
	}
	if len(prs) > 0 {
		parts = append(parts, fmt.Sprintf("%d prs", len(prs)))
	}

	line := ""
	for i, p := range parts {
		if i > 0 {
			line += ", "
		}
		line += p
	}
	line += fmt.Sprintf(" | max %d", config.C.MaxSlots)

	fmt.Println(line)
	return nil
}
