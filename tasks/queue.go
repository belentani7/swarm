package tasks

import (
	"bufio"
	"container/heap"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Priority levels for task scheduling.
const (
	PriorityCritical = 0
	PriorityHigh     = 1
	PriorityNormal   = 2
	PriorityLow      = 3
)

// Task represents a unit of work to assign to an agent.
type Task struct {
	ID          int
	Description string
	Priority    int
	AssignedTo  int  // agent ID, -1 if unassigned
	Completed   bool
}

// Queue is a priority-based task queue.
type Queue struct {
	mu    sync.Mutex
	tasks *taskHeap
	nextID int
}

// LoadFromFile reads tasks from a text file (one per line).
func LoadFromFile(path string) *Queue {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ failed to load tasks from %s: %v\n", path, err)
		return NewQueue()
	}
	defer file.Close()

	q := NewQueue()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		priority := PriorityNormal
		// Parse priority markers
		if strings.HasPrefix(line, "!!!") {
			priority = PriorityCritical
			line = strings.TrimPrefix(line, "!!!")
		} else if strings.HasPrefix(line, "!!") {
			priority = PriorityHigh
			line = strings.TrimPrefix(line, "!!")
		} else if strings.HasPrefix(line, "!") {
			priority = PriorityLow
			line = strings.TrimPrefix(line, "!")
		}

		q.Add(line, priority)
	}

	return q
}

// NewQueue creates an empty task queue.
func NewQueue() *Queue {
	h := &taskHeap{}
	heap.Init(h)
	return &Queue{tasks: h}
}

// Add inserts a task into the queue.
func (q *Queue) Add(description string, priority int) int {
	q.mu.Lock()
	defer q.mu.Unlock()

	id := q.nextID
	q.nextID++

	heap.Push(q.tasks, &Task{
		ID:          id,
		Description: strings.TrimSpace(description),
		Priority:    priority,
		AssignedTo:  -1,
	})

	return id
}

// Next retrieves the highest priority unassigned task.
func (q *Queue) Next() (Task, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.tasks.Len() == 0 {
		return Task{}, false
	}

	task := heap.Pop(q.tasks).(*Task)
	return *task, true
}

// Len returns the number of pending tasks.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.tasks.Len()
}

// MarkCompleted marks a task as done.
func (q *Queue) MarkCompleted(id int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	// Tasks are popped when retrieved, so completion is tracked externally
	// This exists for future persistent queue implementations
}

// --- heap implementation ---

type taskHeap []*Task

func (h taskHeap) Len() int           { return len(h) }
func (h taskHeap) Less(i, j int) bool { return h[i].Priority < h[j].Priority }
func (h taskHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *taskHeap) Push(x interface{}) {
	*h = append(*h, x.(*Task))
}

func (h *taskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return item
}
