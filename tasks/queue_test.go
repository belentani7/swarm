package tasks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStablePriorityOrder(t *testing.T) {
	q := NewQueue()
	q.Add("normal-first", PriorityNormal)
	q.Add("critical-first", PriorityCritical)
	q.Add("normal-second", PriorityNormal)
	q.Add("critical-second", PriorityCritical)
	q.Add("low", PriorityLow)
	q.Add("high", PriorityHigh)
	for _, want := range []string{"critical-first", "critical-second", "high", "normal-first", "normal-second", "low"} {
		got, ok := q.Next()
		if !ok || got.Description != want {
			t.Fatalf("got %q/%v, want %q", got.Description, ok, want)
		}
	}
	if _, ok := q.Next(); ok {
		t.Fatal("expected empty queue")
	}
}

func TestIgnoreBlankTasks(t *testing.T) {
	q := NewQueue()
	if id := q.Add("  ", PriorityCritical); id != -1 {
		t.Fatalf("blank id = %d", id)
	}
	if id := q.Add(" first ", PriorityNormal); id != 0 {
		t.Fatalf("first id = %d", id)
	}
	if q.Len() != 1 {
		t.Fatalf("length = %d", q.Len())
	}
	got, _ := q.Next()
	if got.Description != "first" {
		t.Fatalf("description = %q", got.Description)
	}
}

func TestLoadFromFileMarkers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.txt")
	if err := os.WriteFile(path, []byte("# comment\n!!!\n!!\n!\n!!! urgent\n!! high\nnormal\n! low\n"), 0600); err != nil {
		t.Fatal(err)
	}
	q := LoadFromFile(path)
	if q.Len() != 4 {
		t.Fatalf("length = %d", q.Len())
	}
	for _, want := range []string{"urgent", "high", "normal", "low"} {
		got, ok := q.Next()
		if !ok || got.Description != want {
			t.Fatalf("got %q/%v, want %q", got.Description, ok, want)
		}
	}
}

// A corrupt/truncated task file must not dispatch the valid prefix as if the
// entire plan had been loaded successfully.
func TestLoadFromFileDiscardsPartialQueueOnScannerError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "truncated-tasks.txt")
	data := "!!! task that must not run\n" + strings.Repeat("x", 70*1024) + "\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	q := LoadFromFile(path)
	if q.Len() != 0 {
		t.Fatalf("scanner error must discard all tasks; got %d", q.Len())
	}
}
