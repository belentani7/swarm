#!/usr/bin/env bash
# ═══════════════════════════════════════════════════
# SWARM — One-command installer
# ═══════════════════════════════════════════════════

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# Print banner
cat << 'EOF'

        ███████╗██╗    ██╗ █████╗ ██████╗ ███╗   ███╗
        ██╔════╝██║    ██║██╔══██╗██╔══██╗████╗ ████║
        ███████╗██║ █╗ ██║███████║██████╔╝██╔████╔██║
        ╚════██║██║███╗██║██╔══██║██╔══██╗██║╚██╔╝██║
        ███████║╚███╔███╔╝██║  ██║██║  ██║██║ ╚═╝ ██║
        ╚══════╝ ╚══╝╚══╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝

         ⚡ spawn · orchestrate · monitor · command ⚡

EOF

echo -e "${CYAN}▸ Checking dependencies...${NC}"

# Check Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}✖ Go not found. Install: https://go.dev/dl/${NC}"
    exit 1
fi
GO_VERSION=$(go version | grep -oP '\d+\.\d+')
echo -e "${GREEN}  ✔ Go $(go version | grep -oP 'go[\d.]+')${NC}"

# Check tmux
if ! command -v tmux &> /dev/null; then
    echo -e "${RED}✖ tmux not found.${NC}"
    echo -e "  Install:"
    echo -e "    macOS:   brew install tmux"
    echo -e "    Ubuntu:  sudo apt install tmux"
    echo -e "    Arch:    sudo pacman -S tmux"
    exit 1
fi
echo -e "${GREEN}  ✔ tmux $(tmux -V | grep -oP '[\d.]+')${NC}"

# Check SQLite3 (optional — Go driver bundles it)
if command -v sqlite3 &> /dev/null; then
    echo -e "${GREEN}  ✔ sqlite3 $(sqlite3 --version | cut -d' ' -f1)${NC}"
else
    echo -e "${YELLOW}  ⚠ sqlite3 CLI not found (optional — Go driver bundles it)${NC}"
fi

echo ""
echo -e "${CYAN}▸ Installing swarm...${NC}"

# Install the binary
cd "$(dirname "$0")"
go build -o swarm . 2>/dev/null || {
    echo -e "${YELLOW}  ⚠ local build failed, trying go install...${NC}"
    go install .
}

if [ -f "./swarm" ]; then
    echo -e "${GREEN}  ✔ built ./swarm${NC}"
    # Try to move to a PATH location
    if [ -d "$HOME/.local/bin" ]; then
        cp ./swarm "$HOME/.local/bin/swarm"
        echo -e "${GREEN}  ✔ installed to ~/.local/bin/swarm${NC}"
    elif [ -d "$HOME/bin" ]; then
        cp ./swarm "$HOME/bin/swarm"
        echo -e "${GREEN}  ✔ installed to ~/bin/swarm${NC}"
    else
        echo -e "${YELLOW}  ⚠ no ~/bin or ~/.local/bin — run from current directory: ./swarm${NC}"
    fi
fi

# Create config directory
SWARM_DIR="$HOME/.swarm"
mkdir -p "$SWARM_DIR"

if [ ! -f "$SWARM_DIR/config.yaml" ]; then
    cp config/default.yaml "$SWARM_DIR/config.yaml" 2>/dev/null || {
        cat > "$SWARM_DIR/config.yaml" << 'YAML'
session_name: swarm
agent_cli: claude
budget_limit: 10.0
restart_policy: on-failure
layout: tiled
max_agents: 25
log_lines: 100
YAML
    }
    echo -e "${GREEN}  ✔ created $SWARM_DIR/config.yaml${NC}"
else
    echo -e "${YELLOW}  ⚠ config already exists — skipping${NC}"
fi

echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo -e "${GREEN} ✔ SWARM installed successfully!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo ""
echo -e "  Quick start:"
echo -e "    ${CYAN}swarm up 4${NC}                      # spawn 4 agents"
echo -e "    ${CYAN}swarm up 5 --file tasks.txt${NC}     # spawn with task queue"
echo -e "    ${CYAN}swarm status${NC}                    # live dashboard"
echo -e "    ${CYAN}swarm down${NC}                      # kill everything"
echo ""
echo -e "  Edit config: ${YELLOW}~/.swarm/config.yaml${NC}"
echo ""
