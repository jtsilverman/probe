package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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

	// Make review the default command
	rootCmd.AddCommand(reviewCmd)
	rootCmd.RunE = runReview
}

func runReview(cmd *cobra.Command, args []string) error {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}

	// Determine diff source
	source := "staged"
	if flagBranch != "" {
		source = "branch:" + flagBranch
	} else if flagFile != "" {
		source = "file:" + flagFile
	} else if flagStdin {
		source = "stdin"
	}

	fmt.Fprintf(os.Stderr, "probe: reviewing %s (model: %s)\n", source, flagModel)

	// TODO: implement full pipeline in subsequent tasks
	fmt.Fprintf(os.Stderr, "probe: pipeline not yet wired\n")
	return nil
}
