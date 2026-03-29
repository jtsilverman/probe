package reviewer

const (
	CategoryBug         = "bug"
	CategorySecurity    = "security"
	CategoryPerformance = "performance"
	CategoryStyle       = "style"
	CategoryAIPattern   = "ai-pattern"

	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
)

type Finding struct {
	File        string `json:"file"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
	Code        string `json:"code"`
}

type Review struct {
	Findings []Finding  `json:"findings"`
	Summary  Summary    `json:"summary"`
	Model    string     `json:"model"`
	Tokens   TokenUsage `json:"tokens"`
}

type Summary struct {
	TotalFindings int            `json:"total_findings"`
	BySeverity    map[string]int `json:"by_severity"`
	ByCategory    map[string]int `json:"by_category"`
	Verdict       string         `json:"verdict"`
}

type TokenUsage struct {
	Input  int     `json:"input"`
	Output int     `json:"output"`
	Cost   float64 `json:"cost_usd"`
}

func ComputeSummary(findings []Finding) Summary {
	s := Summary{
		TotalFindings: len(findings),
		BySeverity:    map[string]int{},
		ByCategory:    map[string]int{},
		Verdict:       "pass",
	}
	for _, f := range findings {
		s.BySeverity[f.Severity]++
		s.ByCategory[f.Category]++
	}
	if s.BySeverity[SeverityCritical] > 0 {
		s.Verdict = "fail"
	} else if s.BySeverity[SeverityWarning] > 0 {
		s.Verdict = "warn"
	}
	return s
}
