package diff

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetStagedDiff returns the diff of staged changes
func GetStagedDiff() (string, error) {
	out, err := exec.Command("git", "diff", "--cached").Output()
	if err != nil {
		return "", fmt.Errorf("git diff --cached: %w", err)
	}
	result := string(out)
	if strings.TrimSpace(result) == "" {
		// Fall back to unstaged changes
		out, err = exec.Command("git", "diff").Output()
		if err != nil {
			return "", fmt.Errorf("git diff: %w", err)
		}
		result = string(out)
	}
	return result, nil
}

// GetBranchDiff returns the diff between current HEAD and a branch
func GetBranchDiff(branch string) (string, error) {
	out, err := exec.Command("git", "diff", branch+"...HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git diff %s...HEAD: %w", branch, err)
	}
	return string(out), nil
}

// ReadFile returns the content of a file as a pseudo-diff (all lines as additions)
func ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	lines := strings.Split(string(content), "\n")
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", path, path))
	sb.WriteString("--- /dev/null\n")
	sb.WriteString(fmt.Sprintf("+++ b/%s\n", path))
	sb.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))
	for _, l := range lines {
		sb.WriteString("+" + l + "\n")
	}
	return sb.String(), nil
}

// ReadFullFile reads the current version of a file from the working tree.
// Returns empty string if file doesn't exist or exceeds maxSize.
func ReadFullFile(path string, maxSize int64) string {
	info, err := os.Stat(path)
	if err != nil || info.Size() > maxSize {
		return ""
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}

// ReadStdin reads all of stdin
func ReadStdin() (string, error) {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", fmt.Errorf("no input piped to stdin")
	}
	buf := make([]byte, 0, 1024*1024)
	tmp := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	return string(buf), nil
}
