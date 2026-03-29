package output

import (
	"fmt"
	"strings"

	"github.com/jtsilverman/probe/internal/reviewer"
)

var severityEmoji = map[string]string{
	"critical": "🔴",
	"warning":  "🟡",
	"info":     "🔵",
}

func PrintMarkdown(review *reviewer.Review) {
	if len(review.Findings) == 0 {
		fmt.Println("## ✅ Code Review: No Issues Found")
		fmt.Println()
		printMarkdownSummary(review)
		return
	}

	verdict := "⚠️ Needs Attention"
	if review.Summary.Verdict == "fail" {
		verdict = "❌ Critical Issues Found"
	} else if review.Summary.Verdict == "pass" {
		verdict = "✅ Looks Good"
	}

	fmt.Printf("## %s\n\n", verdict)

	for _, f := range review.Findings {
		emoji := severityEmoji[f.Severity]
		fmt.Printf("### %s %s\n", emoji, f.Title)
		fmt.Printf("**File:** `%s`", f.File)
		if f.StartLine > 0 {
			if f.EndLine > f.StartLine {
				fmt.Printf(" (L%d-%d)", f.StartLine, f.EndLine)
			} else {
				fmt.Printf(" (L%d)", f.StartLine)
			}
		}
		fmt.Println()
		fmt.Printf("**Severity:** %s | **Category:** %s\n\n", f.Severity, f.Category)
		fmt.Println(f.Description)

		if f.Code != "" {
			fmt.Println("\n**Code:**")
			fmt.Println("```")
			fmt.Println(f.Code)
			fmt.Println("```")
		}

		if f.Suggestion != "" {
			fmt.Println("\n**Suggested Fix:**")
			fmt.Println("```")
			fmt.Println(f.Suggestion)
			fmt.Println("```")
		}
		fmt.Println()
		fmt.Println("---")
		fmt.Println()
	}

	printMarkdownSummary(review)
}

func printMarkdownSummary(review *reviewer.Review) {
	s := review.Summary
	parts := []string{}
	if n := s.BySeverity["critical"]; n > 0 {
		parts = append(parts, fmt.Sprintf("%d critical", n))
	}
	if n := s.BySeverity["warning"]; n > 0 {
		parts = append(parts, fmt.Sprintf("%d warnings", n))
	}
	if n := s.BySeverity["info"]; n > 0 {
		parts = append(parts, fmt.Sprintf("%d info", n))
	}

	fmt.Printf("**%d findings:** %s\n\n", s.TotalFindings, strings.Join(parts, ", "))
	fmt.Printf("*Model: %s | Tokens: %d in / %d out | Cost: $%.4f*\n",
		review.Model, review.Tokens.Input, review.Tokens.Output, review.Tokens.Cost)
}
