package reviewer

import (
	"fmt"
	"strings"

	"github.com/jtsilverman/probe/internal/diff"
)

const systemPrompt = `You are a senior software engineer performing a precise code review. You catch issues that linters and type checkers miss: logic bugs, security vulnerabilities, race conditions, missing edge cases, and poor API usage.

You also specifically look for AI-generated code anti-patterns:
- Calls to functions/methods/modules that don't exist in the codebase or standard library (hallucinated APIs)
- Overly broad try/catch or error handling that swallows errors silently
- Variables declared but never used (that look intentional, not accidental)
- Inconsistent error handling (some paths handle errors, others don't)
- Magic numbers or hardcoded values without context
- Defensive code that can't actually fail (checking for null when the type is non-nullable)
- Copy-pasted blocks with subtle inconsistencies between them

For each issue found, respond with a JSON array of findings. Each finding must have:
- "file": exact file path from the diff
- "start_line": line number where the issue starts (from the diff line numbers)
- "end_line": line number where the issue ends
- "severity": "critical" (will cause bugs/security issues), "warning" (should fix), or "info" (style/improvement)
- "category": "bug", "security", "performance", "style", or "ai-pattern"
- "title": short title (under 10 words)
- "description": clear explanation of the issue and why it matters
- "suggestion": suggested fix as code
- "code": the problematic code snippet from the diff

If the code looks good with no issues, return an empty array: []

IMPORTANT: Only report real issues. Do not invent problems. Do not flag standard patterns as issues. Be precise with line numbers.

Respond with ONLY a JSON object in this format, no other text:
{"findings": [...]}`

func BuildReviewPrompt(files []diff.FileDiff) string {
	formatted := diff.FormatForReview(files)

	// Detect primary language
	langCounts := map[string]int{}
	for _, f := range files {
		langCounts[f.Language]++
	}
	primaryLang := "mixed"
	maxCount := 0
	for lang, count := range langCounts {
		if count > maxCount {
			primaryLang = lang
			maxCount = count
		}
	}

	fileList := make([]string, len(files))
	for i, f := range files {
		status := ""
		if f.IsNew {
			status = " [NEW]"
		} else if f.IsDelete {
			status = " [DELETED]"
		}
		fileList[i] = fmt.Sprintf("  - %s (%s)%s", f.Path, f.Language, status)
	}

	return fmt.Sprintf(`Review this code diff. Primary language: %s

Files changed:
%s

Diff (with line numbers):
%s`, primaryLang, strings.Join(fileList, "\n"), formatted)
}

func GetSystemPrompt() string {
	return systemPrompt
}

// EstimateTokens roughly estimates token count (4 chars per token)
func EstimateTokens(text string) int {
	return len(text) / 4
}

// SplitByFile splits a large review into per-file reviews if too large
func SplitByFile(files []diff.FileDiff, maxTokens int) [][]diff.FileDiff {
	total := EstimateTokens(diff.FormatForReview(files))
	if total <= maxTokens {
		return [][]diff.FileDiff{files}
	}

	// Split into individual file batches
	var batches [][]diff.FileDiff
	var current []diff.FileDiff
	currentTokens := 0

	for _, f := range files {
		fileTokens := EstimateTokens(diff.FormatForReview([]diff.FileDiff{f}))
		if currentTokens+fileTokens > maxTokens && len(current) > 0 {
			batches = append(batches, current)
			current = nil
			currentTokens = 0
		}
		current = append(current, f)
		currentTokens += fileTokens
	}
	if len(current) > 0 {
		batches = append(batches, current)
	}

	return batches
}
