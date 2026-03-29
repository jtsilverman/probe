package tests

import (
	"testing"

	"github.com/jtsilverman/probe/internal/diff"
	"github.com/jtsilverman/probe/internal/reviewer"
)

func TestBuildReviewPrompt(t *testing.T) {
	files := []diff.FileDiff{
		{
			Path:     "src/api.py",
			Language: "Python",
			Hunks: []diff.Hunk{
				{
					NewStart: 1,
					Lines: []diff.Line{
						{Type: "add", Content: "import os", NewNum: 1},
						{Type: "add", Content: "SECRET = os.getenv('KEY')", NewNum: 2},
					},
				},
			},
		},
	}

	prompt := reviewer.BuildReviewPrompt(files)

	if len(prompt) == 0 {
		t.Fatal("prompt should not be empty")
	}
	if !contains(prompt, "Python") {
		t.Error("prompt should mention Python")
	}
	if !contains(prompt, "src/api.py") {
		t.Error("prompt should mention file path")
	}
	if !contains(prompt, "import os") {
		t.Error("prompt should contain diff content")
	}
}

func TestSystemPrompt(t *testing.T) {
	prompt := reviewer.GetSystemPrompt()
	if !contains(prompt, "senior software engineer") {
		t.Error("system prompt should describe role")
	}
	if !contains(prompt, "ai-pattern") {
		t.Error("system prompt should mention AI patterns")
	}
	if !contains(prompt, "hallucinated") {
		t.Error("system prompt should mention hallucinated APIs")
	}
	if !contains(prompt, "JSON") {
		t.Error("system prompt should request JSON output")
	}
}

func TestEstimateTokens(t *testing.T) {
	text := "hello world" // 11 chars -> ~2-3 tokens
	tokens := reviewer.EstimateTokens(text)
	if tokens < 1 || tokens > 5 {
		t.Errorf("expected 1-5 tokens for 'hello world', got %d", tokens)
	}
}

func TestSplitByFile(t *testing.T) {
	// Create files that together exceed the limit
	files := make([]diff.FileDiff, 5)
	for i := range files {
		bigContent := make([]diff.Line, 100)
		for j := range bigContent {
			bigContent[j] = diff.Line{Type: "add", Content: "x = " + string(rune('a'+j%26)), NewNum: j + 1}
		}
		files[i] = diff.FileDiff{
			Path:     "file" + string(rune('0'+i)) + ".py",
			Language: "Python",
			Hunks:    []diff.Hunk{{NewStart: 1, Lines: bigContent}},
		}
	}

	// With a very small limit, should split into multiple batches
	batches := reviewer.SplitByFile(files, 100)
	if len(batches) < 2 {
		t.Errorf("expected multiple batches with small token limit, got %d", len(batches))
	}

	// With a huge limit, should be one batch
	batches = reviewer.SplitByFile(files, 1000000)
	if len(batches) != 1 {
		t.Errorf("expected 1 batch with huge limit, got %d", len(batches))
	}
}

func TestComputeSummary(t *testing.T) {
	findings := []reviewer.Finding{
		{Severity: "critical", Category: "security"},
		{Severity: "warning", Category: "bug"},
		{Severity: "warning", Category: "style"},
		{Severity: "info", Category: "style"},
	}

	s := reviewer.ComputeSummary(findings)
	if s.TotalFindings != 4 {
		t.Errorf("expected 4 findings, got %d", s.TotalFindings)
	}
	if s.Verdict != "fail" {
		t.Errorf("expected fail (has critical), got %s", s.Verdict)
	}
	if s.BySeverity["critical"] != 1 {
		t.Errorf("expected 1 critical, got %d", s.BySeverity["critical"])
	}
	if s.ByCategory["style"] != 2 {
		t.Errorf("expected 2 style, got %d", s.ByCategory["style"])
	}
}

func TestComputeSummaryPass(t *testing.T) {
	s := reviewer.ComputeSummary(nil)
	if s.Verdict != "pass" {
		t.Errorf("expected pass for no findings, got %s", s.Verdict)
	}
}

func TestComputeSummaryWarn(t *testing.T) {
	findings := []reviewer.Finding{
		{Severity: "warning", Category: "bug"},
	}
	s := reviewer.ComputeSummary(findings)
	if s.Verdict != "warn" {
		t.Errorf("expected warn, got %s", s.Verdict)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
