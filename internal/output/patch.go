package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/jtsilverman/probe/internal/reviewer"
)

// PrintPatch generates unified diff patches from review findings.
// Each finding with a code+suggestion pair becomes a patch hunk.
func PrintPatch(review *reviewer.Review) {
	patchCount := 0

	for _, f := range review.Findings {
		if f.Suggestion == "" || f.Code == "" {
			continue
		}
		if f.File == "" || f.StartLine == 0 {
			continue
		}

		// Read the actual file to verify the code snippet exists
		content, err := os.ReadFile(f.File)
		if err != nil {
			continue
		}

		fileLines := strings.Split(string(content), "\n")
		codeLines := strings.Split(strings.TrimRight(f.Code, "\n"), "\n")
		sugLines := strings.Split(strings.TrimRight(f.Suggestion, "\n"), "\n")

		// Verify the code snippet matches the file at the specified lines
		startIdx := f.StartLine - 1
		if startIdx < 0 || startIdx+len(codeLines) > len(fileLines) {
			continue
		}

		// Check if the code approximately matches (trim whitespace for comparison)
		matched := true
		for i, cl := range codeLines {
			if strings.TrimSpace(fileLines[startIdx+i]) != strings.TrimSpace(cl) {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}

		// Generate the patch
		fmt.Printf("--- a/%s\n", f.File)
		fmt.Printf("+++ b/%s\n", f.File)
		fmt.Printf("@@ -%d,%d +%d,%d @@\n",
			f.StartLine, len(codeLines),
			f.StartLine, len(sugLines))

		for _, line := range codeLines {
			fmt.Printf("-%s\n", line)
		}
		for _, line := range sugLines {
			fmt.Printf("+%s\n", line)
		}

		patchCount++
	}

	if patchCount == 0 {
		fmt.Fprintf(os.Stderr, "probe: no fixable findings (suggestions must match actual file content)\n")
	} else {
		fmt.Fprintf(os.Stderr, "probe: generated %d patch(es). Apply with: probe --fix | git apply\n", patchCount)
	}
}
