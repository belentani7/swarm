package agent

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/belentani7/swarm/config"
	"github.com/belentani7/swarm/cost"
	"github.com/belentani7/swarm/tasks"
	tmuxmgr "github.com/belentani7/swarm/tmux"
	_ "modernc.org/sqlite"
)

// Status represents an agent's operational state.
type Status string

const (
	StatusSpawning  Status = "spawning"
	StatusActive    Status = "active"
	StatusIdle      Status = "idle"
	StatusDone      Status = "done"
	StatusError     Status = "error"
	StatusStopped   Status = "stopped"
	StatusCrashed   Status = "crashed"
)

// Agent represents a single node in the swarm.
type Agent struct {
	ID         int
	Status     Status
	Task       string
	StartTime  time.Time
	EndTime    time.Time
	Cost       float64
	Tokens     int
	PID        int
	Restarts   int
	mu         sync.RWMutex
}

// AgentState is a snapshot for the dashboard.
type AgentState struct {
	ID       int
	Status   Status
	Task     string
	Cost     float64
	Tokens   int
	Elapsed  time.Duration
	Restarts int
	IsActive bool
}

// Orchestrator manages the swarm of agents.
type Orchestrator struct {
	cfg      *config.Config
	agents   []*Agent
	queue    *tasks.Queue
	tracker  *cost.Tracker
	db       *sql.DB
	mu       sync.RWMutex
	stopCh   chan struct{}
}

// NewOrchestrator creates a new orchestrator with SQLite session tracking.
func NewOrchestrator(cfg *config.Config) (*Orchestrator, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := cfg.DataDir + "/swarm.db"
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := initDB(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	return &Orchestrator{
		cfg:     cfg,
		tracker: cost.NewTracker(cfg.BudgetLimit),
		stopCh:  make(chan struct{}),
		db:      db,
	}, nil
}

func initDB(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		ended_at DATETIME,
		agent_count INTEGER,
		total_cost REAL DEFAULT 0,
		total_tokens INTEGER DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS agents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER,
		node_id INTEGER,
		task TEXT,
		status TEXT DEFAULT 'spawning',
		cost REAL DEFAULT 0,
		tokens INTEGER DEFAULT 0,
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		ended_at DATETIME,
		restarts INTEGER DEFAULT 0,
		FOREIGN KEY(session_id) REFERENCES sessions(id)
	);
	CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER,
		agent_id INTEGER,
		event_type TEXT,
		payload TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(schema)
	return err
}

// SpawnAgents creates N agents with optional task assignment.
func (o *Orchestrator) SpawnAgents(count int, taskTemplate string, taskFile string) []*Agent {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Create session record
	var sessionID int64
	err := o.db.QueryRow("INSERT INTO sessions (agent_count) VALUES (?) RETURNING id", count).Scan(&sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ session tracking failed: %v\n", err)
	}

	// Load tasks if file specified
	if taskFile != "" {
		o.queue = tasks.LoadFromFile(taskFile)
	}

	agents := make([]*Agent, 0, count)
	for i := 0; i < count; i++ {
		task := ""
		if taskTemplate != "" {
			task = strings.ReplaceAll(taskTemplate, "{name}", fmt.Sprintf("node-%d", i))
		}
		if o.queue != nil {
			if t, ok := o.queue.Next(); ok {
				task = t.Description
			}
		}

		a := &Agent{
			ID:        i,
			Status:    StatusSpawning,
			Task:      task,
			StartTime: time.Now(),
		}
		agents = append(agents, a)

		// Persist
		_, _ = o.db.Exec("INSERT INTO agents (session_id, node_id, task, status) VALUES (?, ?, ?, ?)",
			sessionID, i, task, string(StatusSpawning))

		fmt.Printf("  ⚡ node #%d online — %s\n", i, truncate(task, 40))
	}

	o.agents = agents
	return agents
}

// LoadAgentState reconstructs agent state from tmux panes for the dashboard.
func (o *Orchestrator) LoadAgentState(panes []tmuxmgr.PaneInfo) []AgentState {
	o.mu.RLock()
	defer o.mu.RUnlock()

	states := make([]AgentState, len(panes))
	for i, p := range panes {
		// Match with tracked agent if available
		var tracked *Agent
		for _, a := range o.agents {
			if a.ID == p.ID {
				tracked = a
				break
			}
		}

		state := AgentState{
			ID:       p.ID,
			Status:   inferStatus(p),
			IsActive: p.Active,
		}
		if tracked != nil {
			tracked.mu.RLock()
			state.Task = tracked.Task
			state.Cost = tracked.Cost
			state.Tokens = tracked.Tokens
			state.Elapsed = time.Since(tracked.StartTime)
			state.Restarts = tracked.Restarts
			tracked.mu.RUnlock()
		}

		states[i] = state
	}
	return states
}

func inferStatus(p tmuxmgr.PaneInfo) Status {
	cmd := strings.ToLower(p.Command)
	switch {
	case strings.Contains(cmd, "bash") || strings.Contains(cmd, "zsh") || strings.Contains(cmd, "fish"):
		return StatusIdle
	case strings.Contains(cmd, "claude") || strings.Contains(cmd, "aider") || strings.Contains(cmd, "codex"):
		return StatusActive
	default:
		return StatusActive
	}
}

// StartCostTracking begins periodic cost monitoring.
func (o *Orchestrator) StartCostTracking() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				o.tracker.Poll()
				if o.cfg.BudgetLimit > 0 && o.tracker.TotalCost() > o.cfg.BudgetLimit {
					fmt.Printf("\n🚨 Budget limit exceeded! ($%.4f / $%.4f)\n", o.tracker.TotalCost(), o.cfg.BudgetLimit)
					fmt.Println("   Stopping all agents...")
					o.ShutdownAll()
					return
				}
			case <-o.stopCh:
				return
			}
		}
	}()
}

// ShutdownAll terminates all agents and closes the session.
func (o *Orchestrator) ShutdownAll() {
	close(o.stopCh)
	o.mu.Lock()
	defer o.mu.Unlock()

	now := time.Now()
	for _, a := range o.agents {
		a.mu.Lock()
		if a.Status != StatusDone && a.Status != StatusStopped {
			a.Status = StatusStopped
			a.EndTime = now
		}
		a.mu.Unlock()
	}

	// Update session record
	_, _ = o.db.Exec("UPDATE sessions SET ended_at = ?, total_cost = ?, total_tokens = ? WHERE ended_at IS NULL",
		now, o.tracker.TotalCost(), o.tracker.TotalTokens())
}

// Config returns the orchestrator's config.
func (o *Orchestrator) Config() *config.Config {
	return o.cfg
}

// CostReport returns a formatted cost summary.
func (o *Orchestrator) CostReport() cost.Report {
	return o.tracker.Report()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
