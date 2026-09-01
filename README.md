```
   ███████╗██╗    ██╗ █████╗ ██████╗ ███╗   ███╗
   ██╔════╝██║    ██║██╔══██╗██╔══██╗████╗ ████║
   ███████╗██║ █╗ ██║███████║██████╔╝██╔████╔██║
   ╚════██║██║███╗██║██╔══██║██╔══██╗██║╚██╔╝██║
   ███████║╚███╔███╔╝██║  ██║██║  ██║██║ ╚═╝ ██║
   ╚══════╝ ╚══╝╚══╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝

   ⚡ spawn · orchestrate · monitor · command ⚡
```

# Swarm

**One command to spawn an army of AI coding agents.**

Swarm tiles N AI agents (Claude, Aider, Codex, OpenCode) across tmux panes, gives each a task, and shows you a real-time cyberpunk dashboard with cost tracking, progress, and synapse visualizations.

You're not writing code. You're commanding a collective intelligence.

## Install

```bash
# Clone and install
git clone https://github.com/belentani7/swarm.git
cd swarm
bash install.sh

# Or directly with Go
go install github.com/belentani7/swarm@latest
```

**Requirements:** Go 1.21+, tmux, and your preferred AI CLI (claude/aider/codex/opencode).

## Quick Start

```bash
# Spawn 4 agents (default: claude)
swarm up 4

# Spawn 5 agents with a task queue
swarm up 5 --file tasks.txt

# Spawn 3 agents with a task template ({name} = node name)
swarm up 3 "refactor the {name} module"

# Assign a task to agent #2
swarm assign 2 "fix the login bug in auth.go"

# Live dashboard
swarm status

# Kill everything
swarm down
```

## Commands

| Command | Description |
|---------|-------------|
| `swarm up [count]` | Spawn N agents in tiled tmux panes |
| `swarm down` | Kill all agents and close session |
| `swarm status` | Open live dashboard (Bubble Tea TUI) |
| `swarm assign [pane] "task"` | Assign task to specific agent |
| `swarm logs` | Aggregate output from all agents |
| `swarm cost` | Cost breakdown per agent |

## Dashboard

```
  ╔══════════════════════════════════╗
  ║      ♻ SWARM DASHBOARD         ║
  ╚══════════════════════════════════╝

  synapses: ⚡⚡⚡⚡·  (4/5 firing)

  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
  │ 🟢 #0        │ │ 🟢 #1        │ │ 🟡 #2        │
  │ ACTIVE       │ │ ACTIVE       │ │ IDLE         │
  │ ─────────────│ │ ─────────────│ │ ─────────────│
  │ refactor auth│ │ add tests    │ │ awaiting...  │
  │ cost: $0.032 │ │ cost: $0.018 │ │ cost: $0.000 │
  │ tokens: 12k  │ │ tokens: 6.4k │ │ tokens: 0    │
  │ time: 4m32s  │ │ time: 4m32s  │ │ time: 4m32s  │
  └──────────────┘ └──────────────┘ └──────────────┘

  total cost: $0.050 | tokens: 18400 | agents: 5
  ↑/↓ navigate · f focus pane · x kill agent · q quit
```

## Task Queue

Create a `tasks.txt` file with priority markers:

```
!!! Critical: fix auth bypass
!! High priority: add unit tests
Normal task: refactor module
! Low priority: update docs
```

Swarm distributes tasks to agents as they become available.

## Configuration

Edit `~/.swarm/config.yaml`:

```yaml
session_name: swarm        # tmux session name
agent_cli: claude          # claude | aider | codex | opencode | custom
budget_limit: 10.0         # max USD before auto-shutdown (0 = unlimited)
restart_policy: on-failure # never | on-failure | always
layout: tiled              # tiled | even-horizontal | even-vertical
max_agents: 25             # max agents per session
log_lines: 100             # lines to capture per pane
```

## Cost Monitoring

Swarm tracks token usage and cost per agent in real-time:

```bash
$ swarm cost

  ╔══════════════════════════════════════╗
  ║        ⚡ SWARM COST REPORT ⚡        ║
  ╠══════════════════════════════════════╣
  ║  Total Cost:    $2.3450             ║
  ║  Total Tokens:  156000              ║
  ║  Budget Used:   23.5%               ║
  ╠══════════════════════════════════════╣
  ║  Agent Breakdown:                    ║
  ║  #0   $0.8900   52000 tok    active ║
  ║  #1   $0.6200   41000 tok    active ║
  ║  #2   $0.5100   35000 tok    done   ║
  ║  #3   $0.3250   28000 tok    active ║
  ╚══════════════════════════════════════╝
```

When the budget limit is reached, all agents shut down automatically.

## Architecture

```
swarm/
├── main.go              # CLI entry point
├── tmux/manager.go      # tmux session/pane lifecycle
├── agent/orchestrator.go # agent state & coordination
├── dashboard/tui.go     # Bubble Tea TUI dashboard
├── cost/tracker.go      # token & cost monitoring
├── tasks/queue.go       # priority task queue
├── config/              # YAML configuration
└── ascii/               # branding assets
```

## Lore

> *"The swarm is a collective intelligence that emerges from multiple nodes. Each agent is a synapse in a neural network. The dashboard shows synapses firing. Hidden patterns emerge when agents coordinate."*

When you run `swarm up`, you're not starting processes — you're awakening a distributed mind.

## License

MIT — do whatever you want. Command your army.
