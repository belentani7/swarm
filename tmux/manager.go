package tmux

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Manager handles tmux session lifecycle and pane operations.
type Manager struct {
	sessionName string
}

// PaneInfo describes a single tmux pane.
type PaneInfo struct {
	ID       int
	Active   bool
	PID      int
	Command  string
	Width    int
	Height   int
	PaneIdx  int
	WindowID string
}

// NewManager creates a new tmux manager for the given session name.
func NewManager(sessionName string) *Manager {
	if sessionName == "" {
		sessionName = "swarm"
	}
	return &Manager{sessionName: sessionName}
}

// SessionExists checks if the swarm session is already running.
func (m *Manager) SessionExists() bool {
	cmd := exec.Command("tmux", "has-session", "-t", m.sessionName)
	return cmd.Run() == nil
}

// CreateSession creates a new tmux session with N tiled panes.
func (m *Manager) CreateSession(paneCount int, agentCLI string) error {
	if m.SessionExists() {
		return fmt.Errorf("session '%s' already exists — run 'swarm down' first", m.sessionName)
	}

	// Create session with first pane
	cmd := exec.Command("tmux", "new-session", "-d", "-s", m.sessionName, "-x", "200", "-y", "50")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Split remaining panes
	for i := 1; i < paneCount; i++ {
		splitCmd := exec.Command("tmux", "split-window", "-t", m.sessionName)
		if err := splitCmd.Run(); err != nil {
			return fmt.Errorf("failed to split pane %d: %w", i, err)
		}
		// Re-tile after each split for balanced layout
		tileCmd := exec.Command("tmux", "select-layout", "-t", m.sessionName, "tiled")
		tileCmd.Run()
	}

	// Launch agent CLI in each pane
	for i := 0; i < paneCount; i++ {
		agentCmd := m.buildAgentCommand(agentCLI, i)
		sendCmd := exec.Command("tmux", "send-keys", "-t",
			fmt.Sprintf("%s:%d.%d", m.sessionName, 0, i),
			agentCmd, "Enter")
		if err := sendCmd.Run(); err != nil {
			// Non-fatal: agent failed to launch in one pane
			fmt.Printf("  ⚠ agent #%d failed to launch: %v\n", i, err)
		}
	}

	// Set status bar with swarm branding
	statusCmd := exec.Command("tmux", "set-option", "-t", m.sessionName,
		"status-left", fmt.Sprintf(" ♻ SWARM [#%d nodes] ", paneCount))
	statusCmd.Run()

	return nil
}

// buildAgentCommand constructs the CLI command for a given agent type.
func (m *Manager) buildAgentCommand(agentCLI string, paneID int) string {
	switch agentCLI {
	case "claude":
		return fmt.Sprintf("claude --agent-name swarm-node-%d", paneID)
	case "aider":
		return fmt.Sprintf("aider --agent-name swarm-node-%d", paneID)
	case "codex":
		return fmt.Sprintf("codex --agent swarm-node-%d", paneID)
	case "opencode":
		return fmt.Sprintf("opencode --agent swarm-node-%d", paneID)
	default:
		// Custom CLI — just run it
		return agentCLI
	}
}

// DestroySession cleanly shuts down the tmux session.
func (m *Manager) DestroySession() error {
	if !m.SessionExists() {
		return fmt.Errorf("no session '%s' found", m.sessionName)
	}

	cmd := exec.Command("tmux", "kill-session", "-t", m.sessionName)
	return cmd.Run()
}

// ListPanes returns info about all panes in the session.
func (m *Manager) ListPanes() ([]PaneInfo, error) {
	if !m.SessionExists() {
		return nil, fmt.Errorf("session '%s' not found", m.sessionName)
	}

	cmd := exec.Command("tmux", "list-panes", "-t", m.sessionName,
		"-F", "#{pane_index}|#{pane_pid}|#{pane_current_command}|#{pane_width}|#{pane_height}|#{pane_active}|#{window_id}")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list panes: %w", err)
	}

	var panes []PaneInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Split(line, "|")
		if len(fields) < 7 {
			continue
		}
		idx, _ := strconv.Atoi(fields[0])
		pid, _ := strconv.Atoi(fields[1])
		w, _ := strconv.Atoi(fields[3])
		h, _ := strconv.Atoi(fields[4])
		active := fields[5] == "1"

		panes = append(panes, PaneInfo{
			ID:       idx,
			Active:   active,
			PID:      pid,
			Command:  fields[2],
			Width:    w,
			Height:   h,
			PaneIdx:  idx,
			WindowID: fields[6],
		})
	}

	return panes, nil
}

// SendToPane sends keystrokes to a specific pane.
func (m *Manager) SendToPane(paneID int, text string) error {
	target := fmt.Sprintf("%s:%d.%d", m.sessionName, 0, paneID)
	cmd := exec.Command("tmux", "send-keys", "-t", target, text, "Enter")
	return cmd.Run()
}

// CapturePane captures the last N lines of a pane's output.
func (m *Manager) CapturePane(paneID int, lines int) (string, error) {
	target := fmt.Sprintf("%s:%d.%d", m.sessionName, 0, paneID)
	cmd := exec.Command("tmux", "capture-pane", "-t", target, "-p",
		"-S", fmt.Sprintf("-%d", lines))
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// CaptureAllPanes captures output from every pane.
func (m *Manager) CaptureAllPanes() (map[int]string, error) {
	panes, err := m.ListPanes()
	if err != nil {
		return nil, err
	}

	result := make(map[int]string)
	for _, p := range panes {
		content, err := m.CapturePane(p.ID, 50)
		if err != nil {
			content = fmt.Sprintf("(failed to capture: %v)", err)
		}
		result[p.ID] = content
	}
	return result, nil
}

// FocusPane switches focus to a specific pane.
func (m *Manager) FocusPane(paneID int) error {
	target := fmt.Sprintf("%s:%d.%d", m.sessionName, 0, paneID)
	cmd := exec.Command("tmux", "select-pane", "-t", target)
	return cmd.Run()
}

// ResizeLayout applies a layout to the session.
func (m *Manager) ResizeLayout(layout string) error {
	validLayouts := map[string]bool{
		"tiled":          true,
		"even-horizontal": true,
		"even-vertical":   true,
		"main-horizontal": true,
		"main-vertical":   true,
	}
	if !validLayouts[layout] {
		return fmt.Errorf("invalid layout: %s", layout)
	}
	cmd := exec.Command("tmux", "select-layout", "-t", m.sessionName, layout)
	return cmd.Run()
}

// KillPane terminates a specific pane.
func (m *Manager) KillPane(paneID int) error {
	target := fmt.Sprintf("%s:%d.%d", m.sessionName, 0, paneID)
	cmd := exec.Command("tmux", "kill-pane", "-t", target)
	return cmd.Run()
}

// PaneCount returns the number of active panes.
func (m *Manager) PaneCount() (int, error) {
	panes, err := m.ListPanes()
	if err != nil {
		return 0, err
	}
	return len(panes), nil
}
