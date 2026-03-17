package format

import (
	"fmt"
	"strings"
	"time"

	"github.com/abix-/k3sc/internal/types"
)

var Loc, _ = time.LoadLocation("America/New_York")

func FmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.In(Loc).Format("3:04 PM MST")
}

func FmtDuration(start, end *time.Time) string {
	if start == nil {
		return ""
	}
	e := time.Now()
	if end != nil {
		e = *end
	}
	d := e.Sub(*start)
	return fmt.Sprintf("%dm %02ds", int(d.Minutes()), int(d.Seconds())%60)
}

func CountPhases(pods []types.AgentPod) (running, completed, failed int) {
	for _, p := range pods {
		switch p.Phase {
		case types.PhaseRunning, types.PhasePending:
			running++
		case types.PhaseSucceeded:
			completed++
		case types.PhaseFailed:
			failed++
		}
	}
	return
}

func IssueLink(repo types.Repo, number int) string {
	url := fmt.Sprintf("https://github.com/%s/%s/issues/%d", repo.Owner, repo.Name, number)
	text := fmt.Sprintf("#%d", number)
	link := fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
	if len(text) < 7 {
		link += strings.Repeat(" ", 7-len(text))
	}
	return link
}

func PRLink(repo types.Repo, number int) string {
	url := fmt.Sprintf("https://github.com/%s/%s/pull/%d", repo.Owner, repo.Name, number)
	text := fmt.Sprintf("#%d", number)
	link := fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
	if len(text) < 7 {
		link += strings.Repeat(" ", 7-len(text))
	}
	return link
}

func Truncate(s string, max int) string {
	if max < 4 {
		max = 4
	}
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
