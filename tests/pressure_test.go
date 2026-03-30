package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jtsilverman/probe/internal/config"
	"github.com/jtsilverman/probe/internal/diff"
	"github.com/jtsilverman/probe/internal/output"
	"github.com/jtsilverman/probe/internal/reviewer"
)

// ============================================================
// .proberc PRESSURE TESTS
// ============================================================

func TestConfigIgnoreGlobPatterns(t *testing.T) {
	cfg := config.Config{
		Ignore: []string{"*.generated.go", "**/*_test.go", "vendor/**"},
	}

	cases := []struct {
		path   string
		expect bool
	}{
		{"models.generated.go", true},
		{"internal/foo_test.go", true},
		{"foo_test.go", true},
		{"main.go", false},
		{"internal/handler.go", false},
		{"test_helper.go", false}, // not *_test.go
	}

	for _, c := range cases {
		if got := cfg.ShouldIgnore(c.path); got != c.expect {
			t.Errorf("ShouldIgnore(%q) = %v, want %v", c.path, got, c.expect)
		}
	}
}

func TestConfigEmptyIgnoreDoesntPanic(t *testing.T) {
	cfg := config.Config{Ignore: nil}
	// Should not panic
	if cfg.ShouldIgnore("anything.go") {
		t.Error("nil ignore list should not match anything")
	}

	cfg2 := config.Config{Ignore: []string{}}
	if cfg2.ShouldIgnore("anything.go") {
		t.Error("empty ignore list should not match anything")
	}
}

func TestConfigMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".proberc"), []byte("{{{{invalid yaml"), 0644)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	cfg := config.Load()
	// Should return empty config, not crash
	if cfg.Severity != "" {
		t.Error("malformed YAML should return empty config")
	}
}

func TestConfigPartialYAML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".proberc"), []byte("severity: critical\n"), 0644)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	cfg := config.Load()
	if cfg.Severity != "critical" {
		t.Errorf("expected severity 'critical', got %q", cfg.Severity)
	}
	if len(cfg.Ignore) != 0 {
		t.Error("missing fields should default to zero values")
	}
}

func TestConfigCustomRulesWithSpecialChars(t *testing.T) {
	dir := t.TempDir()
	content := `rules:
  - name: "no-console.log"
    description: "Remove console.log statements"
    pattern: "console.log("
    severity: warning
    category: style
`
	os.WriteFile(filepath.Join(dir, ".proberc"), []byte(content), 0644)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	cfg := config.Load()
	if len(cfg.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(cfg.Rules))
	}
	if cfg.Rules[0].Pattern != "console.log(" {
		t.Errorf("pattern should preserve special chars, got %q", cfg.Rules[0].Pattern)
	}
}

// ============================================================
// PATCH OUTPUT PRESSURE TESTS
// ============================================================

func TestPatchMultipleFindings(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.go")
	content := `package main

import "fmt"

func main() {
	fmt.Println("line 6")
	fmt.Println("line 7")
	fmt.Println("line 8")
	x := 42
	fmt.Println(x)
}
`
	os.WriteFile(testFile, []byte(content), 0644)

	review := &reviewer.Review{
		Findings: []reviewer.Finding{
			{
				File:       testFile,
				StartLine:  6,
				EndLine:    6,
				Severity:   "warning",
				Category:   "style",
				Code:       `	fmt.Println("line 6")`,
				Suggestion: `	log.Println("line 6")`,
			},
			{
				File:       testFile,
				StartLine:  7,
				EndLine:    7,
				Severity:   "warning",
				Category:   "style",
				Code:       `	fmt.Println("line 7")`,
				Suggestion: `	log.Println("line 7")`,
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	output.PrintPatch(review)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	result := string(buf[:n])

	// Should generate 2 patches
	count := strings.Count(result, "--- a/")
	if count != 2 {
		t.Errorf("expected 2 patches, got %d", count)
	}
}

func TestPatchCodeMismatchSkipped(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.go")
	os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644)

	review := &reviewer.Review{
		Findings: []reviewer.Finding{
			{
				File:       testFile,
				StartLine:  2,
				EndLine:    2,
				Severity:   "warning",
				Category:   "bug",
				Code:       "this code does not exist in the file",
				Suggestion: "fixed code",
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	output.PrintPatch(review)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	result := string(buf[:n])

	if strings.Contains(result, "--- a/") {
		t.Error("should not generate patch when code doesn't match file")
	}
}

func TestPatchNonexistentFile(t *testing.T) {
	review := &reviewer.Review{
		Findings: []reviewer.Finding{
			{
				File:       "/nonexistent/path/to/file.go",
				StartLine:  1,
				Severity:   "warning",
				Code:       "x := 1",
				Suggestion: "x := 2",
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	output.PrintPatch(review)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	result := string(buf[:n])

	if strings.Contains(result, "--- a/") {
		t.Error("should not generate patch for nonexistent file")
	}
}

func TestPatchStartLineOutOfBounds(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.go")
	os.WriteFile(testFile, []byte("package main\n"), 0644)

	review := &reviewer.Review{
		Findings: []reviewer.Finding{
			{
				File:       testFile,
				StartLine:  999,
				Severity:   "warning",
				Code:       "x := 1",
				Suggestion: "x := 2",
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	output.PrintPatch(review)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	result := string(buf[:n])

	if strings.Contains(result, "--- a/") {
		t.Error("should not generate patch when start_line is out of bounds")
	}
}

// ============================================================
// LANGUAGE PROMPT PRESSURE TESTS
// ============================================================

func TestLanguagePromptsAreDistinct(t *testing.T) {
	langs := []string{"Go", "Python", "JavaScript", "TypeScript", "Rust"}
	prompts := map[string]string{}
	for _, lang := range langs {
		p := reviewer.GetSystemPromptForLanguage(lang)
		prompts[lang] = p
	}

	// Each should be different from the base
	base := reviewer.GetSystemPrompt()
	for _, lang := range langs {
		if prompts[lang] == base {
			t.Errorf("%s prompt should differ from base", lang)
		}
	}

	// Go and Python should differ from each other
	if prompts["Go"] == prompts["Python"] {
		t.Error("Go and Python prompts should be different")
	}
}

func TestLanguagePromptContainsBasePrompt(t *testing.T) {
	base := reviewer.GetSystemPrompt()
	for _, lang := range []string{"Go", "Python", "JavaScript", "Rust"} {
		p := reviewer.GetSystemPromptForLanguage(lang)
		if !strings.Contains(p, base) {
			t.Errorf("%s prompt should contain the base system prompt", lang)
		}
	}
}

func TestBuildPromptDetectsLanguage(t *testing.T) {
	files := []diff.FileDiff{
		{Path: "main.go", Language: "Go", Hunks: []diff.Hunk{{Lines: []diff.Line{{Type: "add", Content: "x := 1", NewNum: 1}}}}},
		{Path: "util.go", Language: "Go", Hunks: []diff.Hunk{{Lines: []diff.Line{{Type: "add", Content: "y := 2", NewNum: 1}}}}},
	}
	prompt := reviewer.BuildReviewPrompt(files)
	if !strings.Contains(prompt, "Primary language: Go") {
		t.Error("prompt should detect Go as primary language")
	}
}

func TestBuildPromptMixedLanguages(t *testing.T) {
	files := []diff.FileDiff{
		{Path: "main.go", Language: "Go", Hunks: []diff.Hunk{{Lines: []diff.Line{{Type: "add", Content: "x", NewNum: 1}}}}},
		{Path: "app.py", Language: "Python", Hunks: []diff.Hunk{{Lines: []diff.Line{{Type: "add", Content: "y", NewNum: 1}}}}},
		{Path: "index.js", Language: "JavaScript", Hunks: []diff.Hunk{{Lines: []diff.Line{{Type: "add", Content: "z", NewNum: 1}}}}},
	}
	prompt := reviewer.BuildReviewPrompt(files)
	// Should pick one primary language (any of the three)
	if !strings.Contains(prompt, "Primary language:") {
		t.Error("prompt should contain primary language")
	}
}

// ============================================================
// CACHE PRESSURE TESTS
// ============================================================

func TestCacheEmptyFindings(t *testing.T) {
	tmpHome := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", orig)

	// Cache empty findings list
	reviewer.SetCached("diff", "model", "prompt", []reviewer.Finding{})

	cached, ok := reviewer.GetCached("diff", "model", "prompt")
	if !ok {
		t.Error("should cache even empty findings")
	}
	if len(cached) != 0 {
		t.Error("cached empty findings should return empty slice")
	}
}

func TestCacheCorruptedFile(t *testing.T) {
	tmpHome := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", orig)

	// Write corrupt data to cache location
	cacheDir := filepath.Join(tmpHome, ".cache", "probe")
	os.MkdirAll(cacheDir, 0755)

	// First cache something valid to get the filename
	reviewer.SetCached("diff", "model", "prompt", []reviewer.Finding{{Title: "test"}})

	// Find the cache file and corrupt it
	entries, _ := os.ReadDir(cacheDir)
	for _, e := range entries {
		os.WriteFile(filepath.Join(cacheDir, e.Name()), []byte("not json{{{"), 0644)
	}

	// Should return cache miss, not crash
	_, ok := reviewer.GetCached("diff", "model", "prompt")
	if ok {
		t.Error("corrupted cache file should return miss, not hit")
	}
}

func TestCacheKeyDiffersOnPromptChange(t *testing.T) {
	tmpHome := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", orig)

	reviewer.SetCached("diff", "model", "prompt-v1", []reviewer.Finding{{Title: "old"}})

	// Same diff + model but different prompt should miss
	_, ok := reviewer.GetCached("diff", "model", "prompt-v2")
	if ok {
		t.Error("different prompt should produce cache miss")
	}
}

func TestCacheLargeFindings(t *testing.T) {
	tmpHome := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", orig)

	// Cache a large number of findings
	findings := make([]reviewer.Finding, 100)
	for i := range findings {
		findings[i] = reviewer.Finding{
			File:        "test.go",
			StartLine:   i + 1,
			Severity:    "warning",
			Category:    "style",
			Title:       strings.Repeat("x", 100),
			Description: strings.Repeat("description ", 50),
		}
	}

	reviewer.SetCached("big-diff", "model", "prompt", findings)

	cached, ok := reviewer.GetCached("big-diff", "model", "prompt")
	if !ok {
		t.Error("should cache large findings")
	}
	if len(cached) != 100 {
		t.Errorf("expected 100 cached findings, got %d", len(cached))
	}
}

// ============================================================
// FULL-FILE CONTEXT PRESSURE TESTS
// ============================================================

func TestReadFullFileRespectsSizeLimit(t *testing.T) {
	dir := t.TempDir()

	// Create a file just under the limit (50KB)
	smallFile := filepath.Join(dir, "small.go")
	os.WriteFile(smallFile, []byte(strings.Repeat("x", 49*1024)), 0644)
	if diff.ReadFullFile(smallFile, 50*1024) == "" {
		t.Error("file under limit should be readable")
	}

	// Create a file over the limit
	bigFile := filepath.Join(dir, "big.go")
	os.WriteFile(bigFile, []byte(strings.Repeat("x", 51*1024)), 0644)
	if diff.ReadFullFile(bigFile, 50*1024) != "" {
		t.Error("file over limit should return empty")
	}
}

func TestReadFullFileNonexistent(t *testing.T) {
	result := diff.ReadFullFile("/nonexistent/file.go", 50*1024)
	if result != "" {
		t.Error("nonexistent file should return empty")
	}
}
