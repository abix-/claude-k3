package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/abix-/k3s-claude/internal/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	loc, _ = time.LoadLocation("America/New_York")

	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	runningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	failedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
	doneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // gray
	humanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("13")) // magenta
	reviewStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14")) // cyan
	readyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	claimedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
)

type Data struct {
	NodeName      string
	NodeVersion   string
	Pods          []types.AgentPod
	Issues        []types.Issue
	DispatcherLog string
}

// GatherFunc is called to refresh data
type GatherFunc func() (*Data, error)

// DispatchFunc is called when user presses 'n'
type DispatchFunc func() (string, error)

type tickMsg time.Time

type Model struct {
	data       *Data
	gatherFn   GatherFunc
	dispatchFn DispatchFunc
	statusMsg  string
	width      int
	height     int
	quitting   bool
}

func NewModel(gatherFn GatherFunc, dispatchFn DispatchFunc) Model {
	return Model{
		gatherFn:   gatherFn,
		dispatchFn: dispatchFn,
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			d, _ := m.gatherFn()
			return d
		},
		tickCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "n":
			m.statusMsg = "dispatching..."
			return m, func() tea.Msg {
				log, err := m.dispatchFn()
				if err != nil {
					return fmt.Sprintf("dispatch error: %v", err)
				}
				return dispatchDone(log)
			}
		case "r":
			m.statusMsg = "refreshing..."
			return m, func() tea.Msg {
				d, _ := m.gatherFn()
				return d
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		return m, tea.Batch(
			func() tea.Msg {
				d, _ := m.gatherFn()
				return d
			},
			tickCmd(),
		)
	case *Data:
		m.data = msg
		if m.statusMsg == "refreshing..." {
			m.statusMsg = ""
		}
	case dispatchDone:
		// refresh after dispatch, keep the dispatch log
		dispLog := string(msg)
		d, _ := m.gatherFn()
		if d != nil {
			d.DispatcherLog = dispLog
			m.data = d
		}
		m.statusMsg = "dispatch complete"
	case string:
		m.statusMsg = msg
	}
	return m, nil
}

type dispatchDone string

func (m Model) View() string {
	if m.quitting || m.data == nil {
		return ""
	}

	var b strings.Builder
	d := m.data
	w := m.width
	if w == 0 {
		w = 120
	}

	// cluster
	b.WriteString(titleStyle.Render("=== CLUSTER ===") + "\n")
	running, completed, failed := countPhases(d.Pods)
	b.WriteString(fmt.Sprintf(" Node: %s %s  |  Agents: %d running, %d completed\n\n",
		d.NodeName, d.NodeVersion, running, completed))

	// issues
	b.WriteString(titleStyle.Render("=== GITHUB ISSUES ===") + "\n")
	if len(d.Issues) == 0 {
		b.WriteString(dimStyle.Render("  (no issues with workflow labels)") + "\n")
	} else {
		b.WriteString(headerStyle.Render(fmt.Sprintf("%-7s %-14s %-10s Title", "Issue", "State", "Owner")) + "\n")
		for _, i := range d.Issues {
			line := fmt.Sprintf("#%-6d %-14s %-10s %s", i.Number, i.State, i.Owner, i.Title)
			switch i.State {
			case "claimed":
				b.WriteString(claimedStyle.Render(line))
			case "needs-human":
				b.WriteString(humanStyle.Render(line))
			case "needs-review":
				b.WriteString(reviewStyle.Render(line))
			case "ready":
				b.WriteString(readyStyle.Render(line))
			default:
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	// dispatcher
	b.WriteString(titleStyle.Render("=== DISPATCHER ===") + "\n")
	if d.DispatcherLog == "" {
		b.WriteString(dimStyle.Render("  (no dispatcher runs found)") + "\n")
	} else {
		for _, line := range strings.Split(strings.TrimSpace(d.DispatcherLog), "\n") {
			b.WriteString(dimStyle.Render("  "+line) + "\n")
		}
	}
	b.WriteString("\n")

	// agents
	b.WriteString(titleStyle.Render(fmt.Sprintf("=== AGENTS (%d running, %d completed, %d failed) ===", running, completed, failed)) + "\n")
	if len(d.Pods) == 0 {
		b.WriteString(dimStyle.Render("  (no agent pods)") + "\n")
	} else {
		b.WriteString(headerStyle.Render(fmt.Sprintf("%-7s %-10s %-11s %-16s %-10s Last Output", "Issue", "Agent", "Status", "Started", "Duration")) + "\n")
		for _, pod := range d.Pods {
			agent := fmt.Sprintf("claude-%d", pod.Slot+types.SlotOffset)
			started := fmtTime(pod.Started)
			duration := fmtDuration(pod.Started, pod.Finished)
			line := fmt.Sprintf("#%-6d %-10s %-11s %-16s %-10s %s",
				pod.Issue, agent, pod.Phase.Display(), started, duration, pod.LogTail)
			switch pod.Phase {
			case types.PhaseRunning, types.PhasePending:
				b.WriteString(runningStyle.Render(line))
			case types.PhaseFailed:
				b.WriteString(failedStyle.Render(line))
			default:
				b.WriteString(doneStyle.Render(line))
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	// help bar
	if m.statusMsg != "" {
		b.WriteString(statusStyle.Render(" "+m.statusMsg))
	} else {
		b.WriteString(dimStyle.Render(" q: quit  |  n: dispatch now  |  r: refresh  |  refreshes every 5s"))
	}
	b.WriteString("\n")

	return b.String()
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.In(loc).Format("3:04 PM MST")
}

func fmtDuration(start, end *time.Time) string {
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

func countPhases(pods []types.AgentPod) (running, completed, failed int) {
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
