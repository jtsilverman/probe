package diff

import (
	"fmt"
	"strconv"
	"strings"
)

type FileDiff struct {
	Path     string
	Language string
	IsNew    bool
	IsDelete bool
	Hunks    []Hunk
	Content  string // Full new content (for new files or --file mode)
}

type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []Line
}

type Line struct {
	Type    string // "add", "remove", "context"
	Content string
	OldNum  int
	NewNum  int
}

func ParseUnifiedDiff(raw string) []FileDiff {
	var files []FileDiff
	var current *FileDiff
	var currentHunk *Hunk

	lines := strings.Split(raw, "\n")
	oldLine, newLine := 0, 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// New file header
		if strings.HasPrefix(line, "diff --git") {
			if current != nil {
				if currentHunk != nil {
					current.Hunks = append(current.Hunks, *currentHunk)
				}
				files = append(files, *current)
			}
			current = &FileDiff{}
			currentHunk = nil
			continue
		}

		if current == nil {
			continue
		}

		// File paths
		if strings.HasPrefix(line, "--- ") {
			old := strings.TrimPrefix(line, "--- ")
			if old == "/dev/null" {
				current.IsNew = true
			}
			continue
		}
		if strings.HasPrefix(line, "+++ ") {
			path := strings.TrimPrefix(line, "+++ ")
			path = strings.TrimPrefix(path, "b/")
			if path == "/dev/null" {
				current.IsDelete = true
			} else {
				current.Path = path
				current.Language = detectLanguage(path)
			}
			continue
		}

		// Binary files
		if strings.HasPrefix(line, "Binary files") {
			continue
		}

		// Hunk header
		if strings.HasPrefix(line, "@@") {
			if currentHunk != nil {
				current.Hunks = append(current.Hunks, *currentHunk)
			}
			h := parseHunkHeader(line)
			currentHunk = &h
			oldLine = h.OldStart
			newLine = h.NewStart
			continue
		}

		// Skip git metadata lines
		if strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "new file") ||
			strings.HasPrefix(line, "deleted file") || strings.HasPrefix(line, "old mode") ||
			strings.HasPrefix(line, "new mode") || strings.HasPrefix(line, "rename") ||
			strings.HasPrefix(line, "similarity") || strings.HasPrefix(line, "dissimilarity") {
			continue
		}

		if currentHunk == nil {
			continue
		}

		// Diff content lines
		if strings.HasPrefix(line, "+") {
			currentHunk.Lines = append(currentHunk.Lines, Line{
				Type:    "add",
				Content: strings.TrimPrefix(line, "+"),
				NewNum:  newLine,
			})
			newLine++
		} else if strings.HasPrefix(line, "-") {
			currentHunk.Lines = append(currentHunk.Lines, Line{
				Type:    "remove",
				Content: strings.TrimPrefix(line, "-"),
				OldNum:  oldLine,
			})
			oldLine++
		} else if strings.HasPrefix(line, " ") || line == "" {
			currentHunk.Lines = append(currentHunk.Lines, Line{
				Type:    "context",
				Content: strings.TrimPrefix(line, " "),
				OldNum:  oldLine,
				NewNum:  newLine,
			})
			oldLine++
			newLine++
		}
	}

	// Don't forget the last file
	if current != nil {
		if currentHunk != nil {
			current.Hunks = append(current.Hunks, *currentHunk)
		}
		if current.Path != "" {
			files = append(files, *current)
		}
	}

	return files
}

func parseHunkHeader(line string) Hunk {
	// @@ -1,5 +1,7 @@
	h := Hunk{}
	parts := strings.Split(line, " ")
	for _, p := range parts {
		if strings.HasPrefix(p, "-") && strings.Contains(p, ",") {
			nums := strings.Split(strings.TrimPrefix(p, "-"), ",")
			if len(nums) == 2 {
				h.OldStart, _ = strconv.Atoi(nums[0])
				h.OldCount, _ = strconv.Atoi(nums[1])
			}
		} else if strings.HasPrefix(p, "+") && strings.Contains(p, ",") {
			nums := strings.Split(strings.TrimPrefix(p, "+"), ",")
			if len(nums) == 2 {
				h.NewStart, _ = strconv.Atoi(nums[0])
				h.NewCount, _ = strconv.Atoi(nums[1])
			}
		} else if strings.HasPrefix(p, "-") && !strings.Contains(p, ",") && p != "-" && p != "---" {
			h.OldStart, _ = strconv.Atoi(strings.TrimPrefix(p, "-"))
			h.OldCount = 1
		} else if strings.HasPrefix(p, "+") && !strings.Contains(p, ",") && p != "+" && p != "+++" {
			h.NewStart, _ = strconv.Atoi(strings.TrimPrefix(p, "+"))
			h.NewCount = 1
		}
	}
	if h.NewStart == 0 {
		h.NewStart = 1
	}
	if h.OldStart == 0 {
		h.OldStart = 1
	}
	return h
}

// FormatForReview creates a line-numbered version of the diff for Claude
func FormatForReview(files []FileDiff) string {
	var sb strings.Builder
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("=== File: %s (%s) ===\n", f.Path, f.Language))
		if f.IsNew {
			sb.WriteString("[NEW FILE]\n")
		}
		if f.IsDelete {
			sb.WriteString("[DELETED FILE]\n")
		}
		for _, h := range f.Hunks {
			for _, l := range h.Lines {
				switch l.Type {
				case "add":
					sb.WriteString(fmt.Sprintf("L%d + %s\n", l.NewNum, l.Content))
				case "remove":
					sb.WriteString(fmt.Sprintf("L%d - %s\n", l.OldNum, l.Content))
				case "context":
					if l.NewNum > 0 {
						sb.WriteString(fmt.Sprintf("L%d   %s\n", l.NewNum, l.Content))
					}
				}
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

var langMap = map[string]string{
	"py": "Python", "js": "JavaScript", "ts": "TypeScript", "tsx": "TypeScript",
	"jsx": "JavaScript", "go": "Go", "rs": "Rust", "rb": "Ruby", "java": "Java",
	"cpp": "C++", "c": "C", "cs": "C#", "swift": "Swift", "kt": "Kotlin",
	"php": "PHP", "sh": "Shell", "bash": "Shell", "yaml": "YAML", "yml": "YAML",
	"json": "JSON", "sql": "SQL", "html": "HTML", "css": "CSS", "md": "Markdown",
}

func detectLanguage(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return "Unknown"
	}
	ext := parts[len(parts)-1]
	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return "Unknown"
}
