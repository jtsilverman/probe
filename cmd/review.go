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
