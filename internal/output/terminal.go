package output

import (
	"fmt"
	"strings"

	"github.com/jtsilverman/probe/internal/reviewer"
)

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	red    = "\033[31m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	green  = "\033[32m"
	dim    = "\033[2m"
	cyan   = "\033[36m"
)

func severityColor(s string) string {
	switch s {
	case "critical":
		return red
	case "warning":
		return yellow
	case "info":
		return blue
	default:
		return ""
	}
}

func severityBadge(s string) string {
	return fmt.Sprintf("%s%s[%s]%s", bold, severityColor(s), strings.ToUpper(s), reset)
}

func categoryTag(c string) string {
	return fmt.Sprintf("%s%s%s", cyan, c, reset)
}

func PrintTerminal(review *reviewer.Review) {
	if len(review.Findings) == 0 {
		fmt.Printf("\n%s%s No issues found %s\n\n", bold, green, reset)
		printSummary(review)
		return
	}

	fmt.Println()
	for i, f := range review.Findings {
		fmt.Printf("%s %s  %s\n", severityBadge(f.Severity), categoryTag(f.Category), f.File)

		if f.StartLine > 0 {
			lineRef := fmt.Sprintf("L%d", f.StartLine)
			if f.EndLine > f.StartLine {
				lineRef = fmt.Sprintf("L%d-%d", f.StartLine, f.EndLine)
			}
			fmt.Printf("  %s%s%s\n", dim, lineRef, reset)
		}

		fmt.Printf("  %s%s%s\n", bold, f.Title, reset)
		fmt.Printf("  %s\n", f.Description)

		if f.Code != "" {
			fmt.Printf("\n  %sCode:%s\n", dim, reset)
			for _, line := range strings.Split(f.Code, "\n") {
				fmt.Printf("  %s  %s%s\n", dim, line, reset)
			}
		}

		if f.Suggestion != "" {
			fmt.Printf("\n  %sFix:%s\n", green, reset)
			for _, line := range strings.Split(f.Suggestion, "\n") {
				fmt.Printf("  %s  %s%s\n", green, line, reset)
			}
		}

		if i < len(review.Findings)-1 {
			fmt.Printf("\n  %s%s%s\n\n", dim, strings.Repeat("-", 60), reset)
		}
	}

	fmt.Println()
	printSummary(review)
}

func printSummary(review *reviewer.Review) {
	s := review.Summary

	verdictColor := green
	if s.Verdict == "fail" {
		verdictColor = red
	} else if s.Verdict == "warn" {
		verdictColor = yellow
	}

	fmt.Printf("%s%s%s%s ", bold, verdictColor, strings.ToUpper(s.Verdict), reset)
	fmt.Printf("%d findings: ", s.TotalFindings)

	parts := []string{}
	if n := s.BySeverity["critical"]; n > 0 {
		parts = append(parts, fmt.Sprintf("%s%d critical%s", red, n, reset))
	}
	if n := s.BySeverity["warning"]; n > 0 {
		parts = append(parts, fmt.Sprintf("%s%d warnings%s", yellow, n, reset))
	}
	if n := s.BySeverity["info"]; n > 0 {
		parts = append(parts, fmt.Sprintf("%s%d info%s", blue, n, reset))
	}
	fmt.Println(strings.Join(parts, ", "))

	fmt.Printf("%sModel: %s | Tokens: %d in / %d out | Cost: $%.4f%s\n\n",
		dim, review.Model, review.Tokens.Input, review.Tokens.Output, review.Tokens.Cost, reset)
}
