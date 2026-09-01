package cost

import (
	"sync"
)

// AgentCost tracks spend for a single agent.
type AgentCost struct {
	ID      int
	Cost    float64
	Tokens  int
	Status  string
}

// Report is the aggregated cost summary.
type Report struct {
	TotalCost     float64
	TotalTokens   int
	BudgetLimit   float64
	BudgetPercent float64
	AgentCosts    []AgentCost
}

// Tracker monitors token usage and cost across all agents.
type Tracker struct {
	mu          sync.RWMutex
	budgetLimit float64
	agentCosts  map[int]*agentCostEntry
	totalCost   float64
	totalTokens int
}

type agentCostEntry struct {
	tokens     int
	cost       float64
	status     string
	lastPolled int64
}

// Pricing tiers (approximate — Claude Sonnet pricing as reference).
const (
	inputPricePer1K  = 0.003  // $3 per 1M tokens → $0.003 per 1K
	outputPricePer1K = 0.015  // $15 per 1M tokens → $0.015 per 1K
)

// NewTracker creates a cost tracker with an optional budget limit.
func NewTracker(budgetLimit float64) *Tracker {
	return &Tracker{
		budgetLimit: budgetLimit,
		agentCosts:  make(map[int]*agentCostEntry),
	}
}

// RecordSpend records token usage for an agent.
func (t *Tracker) RecordSpend(agentID int, inputTokens, outputTokens int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cost := float64(inputTokens)*inputPricePer1K/1000.0 + float64(outputTokens)*outputPricePer1K/1000.0

	entry, exists := t.agentCosts[agentID]
	if !exists {
		entry = &agentCostEntry{}
		t.agentCosts[agentID] = entry
	}

	entry.tokens += inputTokens + outputTokens
	entry.cost += cost
	entry.status = "active"

	t.totalCost += cost
	t.totalTokens += inputTokens + outputTokens
}

// Poll checks for updated costs (placeholder — real impl would read agent logs/API).
func (t *Tracker) Poll() {
	// In a real implementation, this would:
	// 1. Read token usage from each agent's log output
	// 2. Query API usage endpoints if available
	// 3. Update agent cost entries
	// For now, costs are recorded via RecordSpend()
}

// SetStatus updates an agent's cost tracking status.
func (t *Tracker) SetStatus(agentID int, status string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if entry, exists := t.agentCosts[agentID]; exists {
		entry.status = status
	}
}

// TotalCost returns the aggregated cost across all agents.
func (t *Tracker) TotalCost() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.totalCost
}

// TotalTokens returns the total tokens consumed.
func (t *Tracker) TotalTokens() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.totalTokens
}

// BudgetExceeded returns true if spending has exceeded the limit.
func (t *Tracker) BudgetExceeded() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.budgetLimit <= 0 {
		return false
	}
	return t.totalCost > t.budgetLimit
}

// Report generates a full cost breakdown.
func (t *Tracker) Report() Report {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var agentCosts []AgentCost
	for id, entry := range t.agentCosts {
		agentCosts = append(agentCosts, AgentCost{
			ID:     id,
			Cost:   entry.cost,
			Tokens: entry.tokens,
			Status: entry.status,
		})
	}

	budgetPct := 0.0
	if t.budgetLimit > 0 {
		budgetPct = (t.totalCost / t.budgetLimit) * 100
	}

	return Report{
		TotalCost:     t.totalCost,
		TotalTokens:   t.totalTokens,
		BudgetLimit:   t.budgetLimit,
		BudgetPercent: budgetPct,
		AgentCosts:    agentCosts,
	}
}
