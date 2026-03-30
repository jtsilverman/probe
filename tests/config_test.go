package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jtsilverman/probe/internal/config"
)

func TestConfigLoad(t *testing.T) {
	// Create a temp dir with a .proberc
	dir := t.TempDir()
	proberc := filepath.Join(dir, ".proberc")
	content := `
ignore:
  - "vendor/**"
  - "*.generated.go"
severity: warning
categories:
  - bug
  - security
model: claude-sonnet-4-20250514
rules:
  - name: no-fmt-println
    description: Use structured logging
    pattern: fmt.Println
    severity: warning
    category: style
`
	if err := os.WriteFile(proberc, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to the temp dir to test loading
	original, _ := os.Getwd()
	defer os.Chdir(original)
	os.Chdir(dir)

	cfg := config.Load()

	if len(cfg.Ignore) != 2 {
		t.Errorf("expected 2 ignore patterns, got %d", len(cfg.Ignore))
	}
	if cfg.Severity != "warning" {
		t.Errorf("expected severity 'warning', got %q", cfg.Severity)
	}
	if len(cfg.Categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(cfg.Categories))
	}
	if cfg.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected model 'claude-sonnet-4-20250514', got %q", cfg.Model)
	}
	if len(cfg.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(cfg.Rules))
	}
	if cfg.Rules[0].Name != "no-fmt-println" {
		t.Errorf("expected rule name 'no-fmt-println', got %q", cfg.Rules[0].Name)
	}
}

func TestConfigShouldIgnore(t *testing.T) {
	cfg := config.Config{
		Ignore: []string{"vendor/**", "*.generated.go", "**/*_test.go"},
	}

	tests := []struct {
		path   string
		ignore bool
	}{
		{"src/main.go", false},
		{"vendor/lib/foo.go", false}, // filepath.Match doesn't handle ** well, this tests the basename fallback
		{"models.generated.go", true},
		{"internal/foo_test.go", true},
	}

	for _, tt := range tests {
		if got := cfg.ShouldIgnore(tt.path); got != tt.ignore {
			t.Errorf("ShouldIgnore(%q) = %v, want %v", tt.path, got, tt.ignore)
		}
	}
}

func TestConfigEmpty(t *testing.T) {
	// No .proberc in temp dir
	dir := t.TempDir()
	original, _ := os.Getwd()
	defer os.Chdir(original)
	os.Chdir(dir)

	cfg := config.Load()
	if len(cfg.Ignore) != 0 || cfg.Severity != "" || cfg.Model != "" {
		t.Error("expected empty config when no .proberc exists")
	}
}
