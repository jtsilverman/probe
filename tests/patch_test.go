package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jtsilverman/probe/internal/output"
	"github.com/jtsilverman/probe/internal/reviewer"
)

func TestPatchGeneration(t *testing.T) {
	// Create a temp file with known content
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.go")
	content := `package main

import "fmt"

func main() {
	fmt.Println("hello")
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
				Title:      "Use structured logging",
				Code:       `	fmt.Println("hello")`,
				Suggestion: `	log.Println("hello")`,
			},
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	output.PrintPatch(review)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	result := string(buf[:n])

	if !strings.Contains(result, "--- a/") {
		t.Error("patch output should contain --- a/ header")
	}
	if !strings.Contains(result, "+++ b/") {
		t.Error("patch output should contain +++ b/ header")
	}
	if !strings.Contains(result, "@@") {
		t.Error("patch output should contain @@ hunk header")
	}
	if !strings.Contains(result, "-\tfmt.Println") {
		t.Error("patch should contain removed line")
	}
	if !strings.Contains(result, "+\tlog.Println") {
		t.Error("patch should contain added line")
	}
}

func TestPatchNoSuggestion(t *testing.T) {
	review := &reviewer.Review{
		Findings: []reviewer.Finding{
			{
				File:      "test.go",
				StartLine: 1,
				Severity:  "info",
				Category:  "style",
				Title:     "Consider refactoring",
				Code:      "x := 1",
				// No suggestion
			},
		},
	}

	// Should produce no patches (finding has no suggestion)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	output.PrintPatch(review)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	result := string(buf[:n])

	if strings.Contains(result, "---") {
		t.Error("should not generate patch when no suggestion exists")
	}
}
