package output

import (
	"fmt"
	"html"
	"strings"

	"github.com/jtsilverman/probe/internal/reviewer"
)

func PrintHTML(review *reviewer.Review) {
	verdictEmoji := map[string]string{"pass": "&#x2705;", "warn": "&#x26A0;&#xFE0F;", "fail": "&#x274C;"}
	verdictColor := map[string]string{"pass": "#3fb950", "warn": "#d29922", "fail": "#f85149"}
	sevColor := map[string]string{"critical": "#f85149", "warning": "#d29922", "info": "#58a6ff"}

	emoji := verdictEmoji[review.Summary.Verdict]
	color := verdictColor[review.Summary.Verdict]

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Probe Code Review</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #0d1117; color: #e6edf3; padding: 2rem; max-width: 900px; margin: 0 auto; }
h1 { font-size: 1.5rem; margin-bottom: 1rem; }
h2 { font-size: 1.1rem; color: #8b949e; margin: 1.5rem 0 0.75rem; border-bottom: 1px solid #21262d; padding-bottom: 0.5rem; }
.verdict { font-size: 1.3rem; padding: 1rem; border-radius: 8px; background: #161b22; border: 1px solid #30363d; margin-bottom: 1.5rem; }
.finding { background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 1rem; margin-bottom: 1rem; }
.finding-header { display: flex; justify-content: space-between; margin-bottom: 0.5rem; }
.finding-title { font-weight: 600; }
.badge { padding: 2px 8px; border-radius: 4px; font-size: 0.75rem; text-transform: uppercase; }
.file { color: #8b949e; font-size: 0.85rem; margin-bottom: 0.5rem; }
.desc { font-size: 0.9rem; margin-bottom: 0.75rem; line-height: 1.5; }
pre { background: #0d1117; border: 1px solid #30363d; border-radius: 4px; padding: 0.75rem; font-size: 0.8rem; overflow-x: auto; margin: 0.5rem 0; }
code { font-family: 'SF Mono', 'Fira Code', monospace; }
.stats { color: #8b949e; font-size: 0.85rem; margin-top: 1.5rem; }
</style>
</head>
<body>
`)

	sb.WriteString(fmt.Sprintf(`<div class="verdict" style="border-color:%s">
  <span>%s</span> <strong style="color:%s">%s</strong> &mdash; %d finding(s)
</div>
`, color, emoji, color, strings.ToUpper(review.Summary.Verdict), review.Summary.TotalFindings))

	if len(review.Findings) > 0 {
		sb.WriteString("<h2>Findings</h2>\n")
		for _, f := range review.Findings {
			sc := sevColor[f.Severity]
			sb.WriteString(fmt.Sprintf(`<div class="finding">
  <div class="finding-header">
    <span class="finding-title">%s</span>
    <span class="badge" style="background:%s22;color:%s">%s</span>
  </div>
  <div class="file">%s:%d-%d &middot; %s</div>
  <div class="desc">%s</div>
`, html.EscapeString(f.Title), sc, sc, f.Severity, html.EscapeString(f.File), f.StartLine, f.EndLine, f.Category, html.EscapeString(f.Description)))

			if f.Code != "" {
				sb.WriteString(fmt.Sprintf("  <pre><code>%s</code></pre>\n", html.EscapeString(f.Code)))
			}
			if f.Suggestion != "" {
				sb.WriteString(fmt.Sprintf("  <h2 style=\"font-size:0.85rem;border:none;margin:0.5rem 0 0.25rem\">Suggested fix</h2>\n  <pre><code>%s</code></pre>\n", html.EscapeString(f.Suggestion)))
			}
			sb.WriteString("</div>\n")
		}
	}

	sb.WriteString(fmt.Sprintf(`<div class="stats">
  Model: %s | Tokens: %d in / %d out | Cost: $%.4f
</div>
</body>
</html>
`, review.Model, review.Tokens.Input, review.Tokens.Output, review.Tokens.Cost))

	fmt.Print(sb.String())
}
