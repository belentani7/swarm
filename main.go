package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/belentani7/swarm/agent"
	"github.com/belentani7/swarm/config"
	"github.com/belentani7/swarm/dashboard"
	"github.com/belentani7/swarm/tmux"
)

const version = "0.1.0"

func printBanner() {
	fmt.Print(`
   ╔═══════════════════════════════════════════╗
   ║   ▄▀▀▀▀▄  █  █  █ ▄▀▄ █▀▄ █▀▄ █▄ █     ║
   ║   ▀▄▄▄▀  ▀▄▄▀  █▄▀ ▀▄▀ █▀▄ █ █ █ ▀█    ║
   ║                                           ║
   ║   ┌─ command your agent army ─┐          ║
   ║   │  spawn · orchestrate · monitor  │    ║
   ║   └───────────────────────────┘          ║
   ║          v` + version + `                          ║
   ╚═══════════════════════════════════════════╝

`)
}

func printUsage() {
	fmt.Print(`Usage:
  swarm up [count] [task-template]    Spawn N agents in tmux panes
  swarm down                          Kill all agents and close session
  swarm status                        Show live dashboard
  swarm assign [pane] "task"          Assign a task to a specific agent
  swarm logs                          Aggregate logs from all agents
  swarm cost                          Show total cost across all agents
  swarm version                       Print version

Examples:
  swarm up 4                          # spawn 4 idle agents
  swarm up 5 --file tasks.txt         # spawn 5 agents with task queue
  swarm up 3 "refactor the {name} module" # spawn 3 with task template
  swarm assign 2 "fix the login bug" # assign task to agent #2
  swarm cost                          # show spend breakdown

`)
}

func main() {
	if len(os.Args) < 2 {
		printBanner()
		printUsage()
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ config error: %v (using defaults)\n", err)
		cfg = config.Default()
	}

	orchestrator, err := agent.NewOrchestrator(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✖ failed to initialize orchestrator: %v\n", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "up":
		handleUp(orchestrator, cfg)
	case "down":
		handleDown(orchestrator)
	case "status":
		handleStatus(orchestrator)
	case "assign":
		handleAssign(orchestrator)
	case "logs":
		handleLogs(orchestrator)
	case "cost":
		handleCost(orchestrator)
	case "version":
		fmt.Printf("swarm v%s\n", version)
	case "help", "-h", "--help":
		printBanner()
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "✖ unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func handleUp(orch *agent.Orchestrator, cfg *config.Config) {
	count := 3 // default
	if len(os.Args) > 2 {
		n, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "✖ invalid agent count: %s\n", os.Args[2])
			os.Exit(1)
		}
		if n < 1 || n > 50 {
			fmt.Fprintf(os.Stderr, "✖ agent count must be between 1 and 50\n")
			os.Exit(1)
		}
		count = n
	}

	var taskTemplate string
	var taskFile string

	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "--file" && i+1 < len(os.Args) {
			taskFile = os.Args[i+1]
			i++
		} else {
			taskTemplate = os.Args[i]
		}
	}

	printBanner()
	fmt.Printf("⚡ Spawning %d agents...\n\n", count)

	mgr := tmux.NewManager(cfg.SessionName)
	if err := mgr.CreateSession(count, cfg.AgentCLI); err != nil {
		fmt.Fprintf(os.Stderr, "✖ failed to create tmux session: %v\n", err)
		os.Exit(1)
	}

	agents := orch.SpawnAgents(count, taskTemplate, taskFile)
	fmt.Printf("✔ %d agents online. Session: %s\n", len(agents), cfg.SessionName)
	fmt.Printf("  Attach:  tmux attach -t %s\n", cfg.SessionName)
	fmt.Printf("  Status:  swarm status\n")
	fmt.Printf("  Kill:    swarm down\n\n")

	// Start cost tracking
	orch.StartCostTracking()
}

func handleDown(orch *agent.Orchestrator) {
	fmt.Print("⏻ Shutting down the swarm...\n\n")

	mgr := tmux.NewManager(orch.Config().SessionName)
	if err := mgr.DestroySession(); err != nil {
		fmt.Fprintf(os.Stderr, "✖ failed to destroy session: %v\n", err)
	}

	orch.ShutdownAll()
	fmt.Print("✔ All agents terminated. The swarm sleeps.\n")
}

func handleStatus(orch *agent.Orchestrator) {
	mgr := tmux.NewManager(orch.Config().SessionName)
	panes, err := mgr.ListPanes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "✖ no active swarm session found: %v\n", err)
		os.Exit(1)
	}

	agents := orch.LoadAgentState(panes)
	p := dashboard.NewProgram(agents, mgr)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "✖ dashboard error: %v\n", err)
		os.Exit(1)
	}
}

func handleAssign(orch *agent.Orchestrator) {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: swarm assign [pane-id] \"task description\"\n")
		os.Exit(1)
	}

	paneID, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "✖ invalid pane id: %s\n", os.Args[2])
		os.Exit(1)
	}

	task := os.Args[3]
	mgr := tmux.NewManager(orch.Config().SessionName)
	if err := mgr.SendToPane(paneID, task); err != nil {
		fmt.Fprintf(os.Stderr, "✖ failed to assign task: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✔ Assigned to agent #%d: %s\n", paneID, task)
}

func handleLogs(orch *agent.Orchestrator) {
	mgr := tmux.NewManager(orch.Config().SessionName)
	logs, err := mgr.CaptureAllPanes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "✖ failed to capture logs: %v\n", err)
		os.Exit(1)
	}

	for paneID, log := range logs {
		fmt.Printf("\n─── Agent #%d ───────────────────────────────\n", paneID)
		fmt.Println(log)
	}
}

func handleCost(orch *agent.Orchestrator) {
	report := orch.CostReport()
	fmt.Print("\n  ╔══════════════════════════════════════╗\n")
	fmt.Print("  ║        ⚡ SWARM COST REPORT ⚡        ║\n")
	fmt.Print("  ╠══════════════════════════════════════╣\n")
	fmt.Printf("  ║  Total Cost:    $%-18.4f ║\n", report.TotalCost)
	fmt.Printf("  ║  Total Tokens:  %-18d ║\n", report.TotalTokens)
	fmt.Printf("  ║  Budget Used:   %-17s ║\n", fmt.Sprintf("%.1f%%", report.BudgetPercent))
	fmt.Print("  ╠══════════════════════════════════════╣\n")
	fmt.Print("  ║  Agent Breakdown:                    ║\n")
	for _, ac := range report.AgentCosts {
		fmt.Printf("  ║  #%d  $%-8.4f  %-6d tokens  %s ║\n",
			ac.ID, ac.Cost, ac.Tokens, ac.Status)
	}
	fmt.Print("  ╚══════════════════════════════════════╝\n\n")
}
