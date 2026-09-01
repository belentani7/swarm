package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/belentani7/swarm/agent"
	tmuxmgr "github.com/belentani7/swarm/tmux"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Styles for the TUI dashboard.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ff88")).
			BorderStyle(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#00ff88")).
			Padding(0, 1)

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff88")).
			Bold(true)

	idleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffaa00"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff4444")).
			Bold(true)

	doneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#333333")).
			Padding(0, 1).
			Width(30)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Italic(true)

	synapseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ccff"))
)

// tickMsg triggers periodic refresh.
type tickMsg time.Time

// Model is the Bubble Tea model for the dashboard.
type Model struct {
	agents   []agent.AgentState
	manager  *tmuxmgr.Manager
	cursor   int
	width    int
	height   int
	quitting bool
}

// NewProgram creates a new Bubble Tea program for the dashboard.
func NewProgram(agents []agent.AgentState, mgr *tmuxmgr.Manager) *tea.Program {
	m := Model{
		agents:  agents,
		manager: mgr,
	}
	return tea.NewProgram(m)
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.agents)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "f":
			if m.cursor < len(m.agents) {
				m.manager.FocusPane(m.agents[m.cursor].ID)
			}
		case "x":
			if m.cursor < len(m.agents) {
				m.manager.KillPane(m.agents[m.cursor].ID)
			}
		}

	case tickMsg:
		panes, err := m.manager.ListPanes()
		if err == nil {
			states := make([]agent.AgentState, len(panes))
			for i, p := range panes {
				states[i] = agent.AgentState{
					ID:       p.ID,
					Status:   inferStatus(p),
					IsActive: p.Active,
				}
				if i < len(m.agents) {
					states[i].Task = m.agents[i].Task
					states[i].Cost = m.agents[i].Cost
					states[i].Tokens = m.agents[i].Tokens
					states[i].Elapsed = m.agents[i].Elapsed
					states[i].Restarts = m.agents[i].Restarts
				}
			}
			m.agents = states
		}
		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m Model) render() string {
	if m.quitting {
		return "♻ Swarm dashboard closed. Your agents are still running.\n"
	}

	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("♻ SWARM DASHBOARD"))
	b.WriteString("\n\n")

	// Synapse visualization
	active := 0
	for _, a := range m.agents {
		if a.Status == agent.StatusActive {
			active++
		}
	}
	synapses := strings.Repeat("⚡", active) + strings.Repeat("·", len(m.agents)-active)
	b.WriteString(synapseStyle.Render(fmt.Sprintf("  synapses: %s  (%d/%d firing)", synapses, active, len(m.agents))))
	b.WriteString("\n\n")

	// Agent grid
	cols := 3
	if len(m.agents) <= 4 {
		cols = 2
	}
	if len(m.agents) > 9 {
		cols = 4
	}

	rows := (len(m.agents) + cols - 1) / cols
	for r := 0; r < rows; r++ {
		var rowCards []string
		for c := 0; c < cols; c++ {
			idx := r*cols + c
			if idx >= len(m.agents) {
				break
			}
			rowCards = append(rowCards, m.renderAgent(m.agents[idx], idx == m.cursor))
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, rowCards...))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	totalCost := 0.0
	totalTokens := 0
	for _, a := range m.agents {
		totalCost += a.Cost
		totalTokens += a.Tokens
	}
	b.WriteString(fmt.Sprintf("  total cost: $%.4f | tokens: %d | agents: %d\n",
		totalCost, totalTokens, len(m.agents)))
	b.WriteString(helpStyle.Render("  ↑/↓ navigate · f focus pane · x kill agent · q quit"))
	b.WriteString("\n")

	return b.String()
}

func (m Model) renderAgent(a agent.AgentState, selected bool) string {
	statusIcon := getStatusIcon(a.Status)
	statusText := getStatusStyle(a.Status).Render(string(a.Status))

	task := a.Task
	if task == "" {
		task = "awaiting orders..."
	}
	if len(task) > 25 {
		task = task[:22] + "..."
	}

	elapsed := formatDuration(a.Elapsed)

	content := fmt.Sprintf(
		" %s #%d\n %s\n ─────────────\n task: %s\n cost: $%.4f\n tokens: %d\n time: %s\n restarts: %d",
		statusIcon, a.ID, statusText, task, a.Cost, a.Tokens, elapsed, a.Restarts,
	)

	style := cardStyle
	if selected {
		style = style.BorderForeground(lipgloss.Color("#00ff88"))
	}

	return style.Render(content)
}

func getStatusIcon(s agent.Status) string {
	switch s {
	case agent.StatusActive:
		return "🟢"
	case agent.StatusIdle:
		return "🟡"
	case agent.StatusError:
		return "🔴"
	case agent.StatusDone:
		return "⚪"
	case agent.StatusCrashed:
		return "💀"
	case agent.StatusSpawning:
		return "⚡"
	default:
		return "⚫"
	}
}

func getStatusStyle(s agent.Status) lipgloss.Style {
	switch s {
	case agent.StatusActive:
		return activeStyle
	case agent.StatusIdle:
		return idleStyle
	case agent.StatusError, agent.StatusCrashed:
		return errorStyle
	case agent.StatusDone, agent.StatusStopped:
		return doneStyle
	default:
		return idleStyle
	}
}

func inferStatus(p tmuxmgr.PaneInfo) agent.Status {
	cmd := strings.ToLower(p.Command)
	switch {
	case strings.Contains(cmd, "bash") || strings.Contains(cmd, "zsh") || strings.Contains(cmd, "fish"):
		return agent.StatusIdle
	case strings.Contains(cmd, "claude") || strings.Contains(cmd, "aider") || strings.Contains(cmd, "codex"):
		return agent.StatusActive
	default:
		return agent.StatusActive
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
