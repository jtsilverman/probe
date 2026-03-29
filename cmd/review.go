package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jtsilverman/probe/internal/diff"
	"github.com/jtsilverman/probe/internal/output"
	"github.com/jtsilverman/probe/internal/reviewer"
)

var (
	flagBranch   string
	flagFile     string
	flagStdin    bool
	flagJSON     bool
	flagMarkdown bool
	flagFix      bool
	flagSeverity string
	flagCategory string
	flagModel    string
	flagMaxFiles int
	flagCLI      bool
	flagVerify   bool
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review code changes for bugs, security issues, and AI-generated anti-patterns",
	RunE:  runReview,
}

func init() {
	reviewCmd.Flags().StringVar(&flagBranch, "branch", "", "Review diff against branch (e.g., main)")
	reviewCmd.Flags().StringVar(&flagFile, "file", "", "Review a specific file")
	reviewCmd.Flags().BoolVar(&flagStdin, "stdin", false, "Read diff from stdin")
	reviewCmd.Flags().BoolVar(&flagJSON, "json", false, "Output as JSON")
	reviewCmd.Flags().BoolVar(&flagMarkdown, "markdown", false, "Output as GitHub markdown")
	reviewCmd.Flags().BoolVar(&flagFix, "fix", false, "Include fix suggestions")
	reviewCmd.Flags().StringVar(&flagSeverity, "severity", "info", "Minimum severity (info, warning, critical)")
	reviewCmd.Flags().StringVar(&flagCategory, "category", "", "Filter categories (comma-separated: bug,security,performance,style,ai-pattern)")
	reviewCmd.Flags().StringVar(&flagModel, "model", "claude-sonnet-4-20250514", "Claude model to use")
	reviewCmd.Flags().IntVar(&flagMaxFiles, "max-files", 20, "Max files to review")
	reviewCmd.Flags().BoolVar(&flagCLI, "cli", false, "Use claude CLI instead of API (uses your subscription, $0 cost)")
	reviewCmd.Flags().BoolVar(&flagVerify, "verify", false, "After review, re-review the same code to verify findings are real (reduces false positives)")

	rootCmd.AddCommand(reviewCmd)

	// Copy flags to root command so `probe --file` works without `probe review --file`
	rootCmd.Flags().StringVar(&flagBranch, "branch", "", "Review diff against branch (e.g., main)")
	rootCmd.Flags().StringVar(&flagFile, "file", "", "Review a specific file")
	rootCmd.Flags().BoolVar(&flagStdin, "stdin", false, "Read diff from stdin")
	rootCmd.Flags().BoolVar(&flagJSON, "json", false, "Output as JSON")
	rootCmd.Flags().BoolVar(&flagMarkdown, "markdown", false, "Output as GitHub markdown")
	rootCmd.Flags().BoolVar(&flagFix, "fix", false, "Include fix suggestions")
	rootCmd.Flags().StringVar(&flagSeverity, "severity", "info", "Minimum severity")
	rootCmd.Flags().StringVar(&flagCategory, "category", "", "Filter categories")
	rootCmd.Flags().StringVar(&flagModel, "model", "claude-sonnet-4-20250514", "Claude model")
	rootCmd.Flags().IntVar(&flagMaxFiles, "max-files", 20, "Max files to review")
	rootCmd.Flags().BoolVar(&flagCLI, "cli", false, "Use claude CLI instead of API ($0 cost)")
	rootCmd.Flags().BoolVar(&flagVerify, "verify", false, "Re-review to verify findings (reduces false positives)")
	rootCmd.RunE = runReview
}

func runReview(cmd *cobra.Command, args []string) error {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" && !flagCLI {
		return fmt.Errorf("ANTHROPIC_API_KEY environment variable is required.\nGet one at https://console.anthropic.com/\nOr use --cli to use your Claude Code subscription instead ($0 cost).")
	}

	// Get the diff
	var rawDiff string
	var err error
	var source string

	if flagStdin {
		source = "stdin"
		rawDiff, err = diff.ReadStdin()
	} else if flagFile != "" {
		source = "file:" + flagFile
		rawDiff, err = diff.ReadFile(flagFile)
	} else if flagBranch != "" {
		source = "branch:" + flagBranch
		rawDiff, err = diff.GetBranchDiff(flagBranch)
	} else {
		source = "staged changes"
		rawDiff, err = diff.GetStagedDiff()
	}

	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	if strings.TrimSpace(rawDiff) == "" {
		fmt.Fprintf(os.Stderr, "probe: no changes to review\n")
		return nil
	}

	// Parse the diff
	files := diff.ParseUnifiedDiff(rawDiff)
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "probe: no supported files in diff\n")
		return nil
	}

	fmt.Fprintf(os.Stderr, "probe: reviewing %s (%d files, model: %s)\n", source, len(files), flagModel)

	// Run the review
	cfg := reviewer.ReviewConfig{
		APIKey:   apiKey,
		Model:    flagModel,
		MaxFiles: flagMaxFiles,
		UseCLI:   flagCLI,
	}

	review, err := reviewer.RunReview(files, cfg)
	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	// Verify: re-run review and only keep findings confirmed in both passes
	if flagVerify && len(review.Findings) > 0 {
		fmt.Fprintf(os.Stderr, "probe: verifying %d findings (pass 2)...\n", len(review.Findings))
		review2, err := reviewer.RunReview(files, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "probe: verification pass failed: %v (using original results)\n", err)
		} else {
			confirmed := verifyFindings(review.Findings, review2.Findings)
			dropped := len(review.Findings) - len(confirmed)
			if dropped > 0 {
				fmt.Fprintf(os.Stderr, "probe: dropped %d unconfirmed findings, %d verified\n", dropped, len(confirmed))
			} else {
				fmt.Fprintf(os.Stderr, "probe: all %d findings confirmed\n", len(confirmed))
			}
			review.Findings = confirmed
			review.Summary = reviewer.ComputeSummary(confirmed)
			review.Tokens.Input += review2.Tokens.Input
			review.Tokens.Output += review2.Tokens.Output
			review.Tokens.Cost += review2.Tokens.Cost
		}
	}

	// Filter by severity
	if flagSeverity != "info" {
		review.Findings = filterBySeverity(review.Findings, flagSeverity)
		review.Summary = reviewer.ComputeSummary(review.Findings)
	}

	// Filter by category
	if flagCategory != "" {
		cats := strings.Split(flagCategory, ",")
		review.Findings = filterByCategory(review.Findings, cats)
		review.Summary = reviewer.ComputeSummary(review.Findings)
	}

	// Output
	if flagJSON {
		output.PrintJSON(review)
	} else if flagMarkdown {
		output.PrintMarkdown(review)
	} else {
		output.PrintTerminal(review)
	}

	// Exit code
	if review.Summary.Verdict == "fail" {
		os.Exit(1)
	}

	return nil
}

func filterBySeverity(findings []reviewer.Finding, minSeverity string) []reviewer.Finding {
	severityOrder := map[string]int{"info": 0, "warning": 1, "critical": 2}
	minLevel := severityOrder[minSeverity]

	var filtered []reviewer.Finding
	for _, f := range findings {
		if severityOrder[f.Severity] >= minLevel {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// verifyFindings keeps only findings from pass 1 that are confirmed by pass 2.
// A finding is "confirmed" if pass 2 has a finding in the same file with matching
// category and overlapping line range (exact title match not required since Claude
// may phrase it differently).
func verifyFindings(pass1, pass2 []reviewer.Finding) []reviewer.Finding {
	var confirmed []reviewer.Finding
	for _, f1 := range pass1 {
		for _, f2 := range pass2 {
			if f1.File == f2.File && f1.Category == f2.Category && linesOverlap(f1, f2) {
				confirmed = append(confirmed, f1)
				break
			}
		}
	}
	return confirmed
}

func linesOverlap(a, b reviewer.Finding) bool {
	aStart, aEnd := a.StartLine, a.EndLine
	bStart, bEnd := b.StartLine, b.EndLine
	if aEnd == 0 {
		aEnd = aStart
	}
	if bEnd == 0 {
		bEnd = bStart
	}
	// Allow some slack (within 5 lines) since Claude may reference slightly different ranges
	return aStart <= bEnd+5 && bStart <= aEnd+5
}

func filterByCategory(findings []reviewer.Finding, categories []string) []reviewer.Finding {
	catSet := map[string]bool{}
	for _, c := range categories {
		catSet[strings.TrimSpace(c)] = true
	}

	var filtered []reviewer.Finding
	for _, f := range findings {
		if catSet[f.Category] {
			filtered = append(filtered, f)
		}
	}
	return filtered
}
