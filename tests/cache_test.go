package tests

import (
	"os"
	"testing"

	"github.com/jtsilverman/probe/internal/reviewer"
)

func TestCacheSetAndGet(t *testing.T) {
	// Use a temp home dir
	tmpHome := t.TempDir()
	original := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", original)

	diff := "some diff content"
	model := "claude-sonnet-4"
	prompt := "system prompt"

	findings := []reviewer.Finding{
		{
			File:      "test.go",
			StartLine: 1,
			Severity:  "warning",
			Category:  "bug",
			Title:     "Test finding",
		},
	}

	// Should not be cached initially
	_, ok := reviewer.GetCached(diff, model, prompt)
	if ok {
		t.Error("expected cache miss on first call")
	}

	// Cache it
	reviewer.SetCached(diff, model, prompt, findings)

	// Should be cached now
	cached, ok := reviewer.GetCached(diff, model, prompt)
	if !ok {
		t.Error("expected cache hit after SetCached")
	}
	if len(cached) != 1 {
		t.Errorf("expected 1 finding, got %d", len(cached))
	}
	if cached[0].Title != "Test finding" {
		t.Errorf("expected title 'Test finding', got %q", cached[0].Title)
	}
}

func TestCacheMissOnDifferentInput(t *testing.T) {
	tmpHome := t.TempDir()
	original := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", original)

	reviewer.SetCached("diff1", "model", "prompt", []reviewer.Finding{})

	// Different diff should miss
	_, ok := reviewer.GetCached("diff2", "model", "prompt")
	if ok {
		t.Error("expected cache miss for different diff content")
	}

	// Different model should miss
	_, ok = reviewer.GetCached("diff1", "other-model", "prompt")
	if ok {
		t.Error("expected cache miss for different model")
	}
}
