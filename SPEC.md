# Probe -- AI Code Review CLI

## Overview

Zero-config CLI that reviews code diffs using Claude and catches issues linters miss: logic bugs, security holes, missing edge cases, and patterns specific to AI-generated code (hallucinated APIs, overly defensive error handling, copy-paste inconsistencies). Point it at a git diff, staged changes, or a file and get a structured review in seconds. CodeRabbit and Diffray exist but require accounts/config. Probe is zero-config, single binary, BYO API key, with CI-ready JSON output.

## Scope

- **Timebox:** 2 days
- **Building:**
  - CLI that reviews: git staged changes, git diff between branches, specific files, or piped input
  - Claude API integration for semantic code review (tool use not needed, just structured prompting)
  - Review categories: bugs, security, performance, style, ai-patterns (AI-specific anti-patterns)
  - Severity levels: critical, warning, info
  - Output formats: colored terminal (default), JSON (for CI), GitHub-compatible markdown
  - Per-finding: file, line range, severity, category, description, suggested fix
  - AI-code specific detections: hallucinated function calls (calls to functions not defined in the diff or imports), overly broad try/catch, unused variables that look intentional, inconsistent error handling patterns, magic numbers without context
  - Config-free by default, optional .proberc for custom rules/ignore patterns
  - `--fix` flag that outputs a patch with suggested fixes
  - Exit code 1 if any critical findings (for CI gates)
- **Not building:** GitHub/GitLab integration, PR comments, web UI, auto-apply fixes, local model support (MVP is Claude-only), multi-repo analysis
- **Ship target:** GitHub + go install

## Project Type

**Agent-based** -- structured prompting with Claude API, no tool use needed

## Stack

- **Language:** Go (single binary, no runtime deps, fast startup for CI)
- **Key packages:** anthropics/anthropic-sdk-go (official Claude SDK, 936*), lipgloss (terminal styling), cobra (CLI)
- **Why Go:** Single binary distribution is critical for a CI tool. Users do `go install` or download a binary. No Python/Node runtime. Fast cold start. Third Go project but different domain (CLI dev tool vs web server vs skill scorer).

## Architecture

### File Structure
```
probe/
  main.go               -- CLI entry point (cobra)
  cmd/
    review.go            -- Review command (default)
    version.go           -- Version command
  internal/
    diff/
      parser.go          -- Parse git diffs into structured hunks
      git.go             -- Git operations (staged, branch diff, file read)
    reviewer/
      reviewer.go        -- Core review logic, Claude API calls
      prompt.go          -- System prompt + review prompt construction
      categories.go      -- Review category definitions
    output/
      terminal.go        -- Colored terminal output
      json.go            -- JSON output
      markdown.go        -- GitHub-compatible markdown
    config/
      config.go          -- Optional .proberc loading
  go.mod
  go.sum
  README.md
  tests/
    fixtures/            -- Sample diffs for testing
    reviewer_test.go
    diff_test.go
    output_test.go
```

### Data Model
```go
type ReviewRequest struct {
    Diff     string   // Raw diff text
    Files    []string // File paths involved
    Language string   // Detected language
    Context  string   // Optional: full file content for context
}

type Review struct {
    Findings []Finding `json:"findings"`
    Summary  Summary   `json:"summary"`
    Model    string    `json:"model"`
    Tokens   TokenUsage `json:"tokens"`
}

type Finding struct {
    File        string `json:"file"`
    StartLine   int    `json:"start_line"`
    EndLine     int    `json:"end_line"`
    Severity    string `json:"severity"`    // critical, warning, info
    Category    string `json:"category"`    // bug, security, performance, style, ai-pattern
    Title       string `json:"title"`
    Description string `json:"description"`
    Suggestion  string `json:"suggestion"`  // Suggested fix
    Code        string `json:"code"`        // The problematic code snippet
}

type Summary struct {
    TotalFindings int            `json:"total_findings"`
    BySeverity    map[string]int `json:"by_severity"`
    ByCategory    map[string]int `json:"by_category"`
    Verdict       string         `json:"verdict"` // "pass", "warn", "fail"
}

type TokenUsage struct {
    Input  int     `json:"input"`
    Output int     `json:"output"`
    Cost   float64 `json:"cost_usd"`
}
```

### CLI Interface
```
probe [command] [flags]

Commands:
  review    Review code changes (default command)
  version   Print version

Review flags:
  (no args)           Review staged changes (git diff --cached)
  --branch <branch>   Review diff against branch (e.g., main)
  --file <path>       Review a specific file
  --stdin             Read diff from stdin (pipe-friendly)
  --json              Output as JSON
  --markdown          Output as GitHub markdown
  --fix               Include fix suggestions as unified diff patches
  --severity <level>  Minimum severity to report (info, warning, critical)
  --category <cats>   Filter categories (bug,security,performance,style,ai-pattern)
  --model <model>     Claude model to use (default: claude-sonnet-4-20250514)
  --max-files <n>     Max files to review (default: 20)

Environment:
  ANTHROPIC_API_KEY   Required. Claude API key.

Exit codes:
  0  No critical findings
  1  Critical findings found (use in CI: probe || exit 1)
```

### Claude Prompt Strategy

**System prompt:**
```
You are a senior software engineer performing a code review. You catch issues that linters and type checkers miss: logic bugs, security vulnerabilities, race conditions, missing edge cases, and poor API usage.

You also specifically look for AI-generated code anti-patterns:
- Calls to functions/methods that don't exist in the codebase or standard library
- Overly broad try/catch that swallows errors silently
- Variables declared but never used (that look intentional, not accidental)
- Inconsistent error handling (some paths handle errors, others don't)
- Magic numbers or hardcoded values without context
- Defensive code that can't actually fail (checking for null when the type is non-nullable)
- Copy-pasted blocks with subtle inconsistencies

For each issue found, provide:
1. The exact file and line range
2. Severity: critical (will cause bugs/security issues), warning (should fix), info (style/improvement)
3. Category: bug, security, performance, style, ai-pattern
4. A clear description of the issue
5. A suggested fix (as code)

If the code looks good, say so. Don't invent issues.
Respond in JSON format matching this schema: [schema]
```

**Review prompt:**
```
Review this code diff. The language is {language}.

{diff}

{optional: full file context for files that were modified}
```

## Task List

### Phase 1: Project Setup + Diff Parsing

#### Task 1.1: Project scaffold and CLI
**Files:** `main.go` (create), `cmd/review.go` (create), `cmd/version.go` (create), `go.mod` (create)
**Do:** Initialize Go module as `github.com/jtsilverman/probe`. Set up cobra CLI with review as default command. Parse all flags (--json, --markdown, --fix, --branch, --file, --stdin, --severity, --category, --model, --max-files). Print parsed flags and exit. Read ANTHROPIC_API_KEY from env.
**Validate:** `go build && ./probe --help` shows usage, `./probe version` prints version

#### Task 1.2: Git diff parser
**Files:** `internal/diff/parser.go` (create), `internal/diff/git.go` (create), `tests/diff_test.go` (create)
**Do:** Parse unified diff format into structured hunks: file path, old/new line numbers, added/removed/context lines. Git operations: get staged diff (`git diff --cached`), branch diff (`git diff <branch>...HEAD`), read file content. Detect language from file extension. Handle binary files (skip). Handle new files, deleted files, renamed files.
**Validate:** `go test ./tests/ -run TestDiff -v` passes with fixture diffs

### Phase 2: Claude Review Engine

#### Task 2.1: Review prompt construction
**Files:** `internal/reviewer/prompt.go` (create), `internal/reviewer/categories.go` (create)
**Do:** Build the system prompt and review prompt. Define category constants (bug, security, performance, style, ai-pattern) and severity levels. Construct the review request with diff content and optional full-file context. Define the JSON response schema for Claude to follow. Handle large diffs by splitting into per-file reviews if total tokens would exceed model limits (estimate 4 chars per token, cap at 150K chars).
**Validate:** `go test ./tests/ -run TestPrompt -v` -- test prompt construction with sample diffs

#### Task 2.2: Claude API integration
**Files:** `internal/reviewer/reviewer.go` (create), `tests/reviewer_test.go` (create)
**Do:** Call Claude API with the constructed prompt. Parse the JSON response into Review struct. Handle API errors gracefully (rate limits, invalid key, model errors). Track token usage and calculate cost (Sonnet: $3/$15 per MTok). Support --model flag for model override. Retry once on 5xx errors.
**Validate:** `go test ./tests/ -run TestReviewer -v` -- test with mocked HTTP responses

### Phase 3: Output Formatting

#### Task 3.1: Terminal output
**Files:** `internal/output/terminal.go` (create), `tests/output_test.go` (create)
**Do:** Colored terminal output using lipgloss. Red for critical, yellow for warning, blue for info. Show file:line, severity badge, category tag, description, code snippet, and suggested fix. Summary at the end: total findings by severity, verdict (pass/warn/fail), token usage and cost.
**Validate:** `go test ./tests/ -run TestTerminalOutput -v`

#### Task 3.2: JSON and markdown output
**Files:** `internal/output/json.go` (create), `internal/output/markdown.go` (create)
**Do:** JSON: serialize Review struct directly. Markdown: GitHub-compatible format with headers, code blocks, severity badges as emojis. Both write to stdout.
**Validate:** `go test ./tests/ -run TestJSONOutput -v && go test ./tests/ -run TestMarkdownOutput -v`

### Phase 4: Integration + Polish

#### Task 4.1: Wire everything together
**Files:** `cmd/review.go` (modify), `internal/config/config.go` (create)
**Do:** Connect the full pipeline: parse CLI flags -> get diff (staged/branch/file/stdin) -> parse diff -> build prompt -> call Claude -> parse response -> format output -> exit code. Load optional .proberc (JSON: custom rules to add/ignore, severity threshold). Handle edge cases: no staged changes, empty diff, API key missing.
**Validate:** Create a test file with a deliberate bug, stage it, run `./probe` and verify it catches the bug. `echo "func main() { fmt.Println(undefined_var) }" | ./probe --stdin --json` produces valid JSON with a finding.

#### Task 4.2: Test fixtures and integration tests
**Files:** `tests/fixtures/` (create), `tests/integration_test.go` (create)
**Do:** Create fixture diffs with known issues: security vulnerability (SQL injection), logic bug (off-by-one), AI pattern (hallucinated function call), performance issue (N+1 query pattern), clean code (no issues). Integration test: run full pipeline against each fixture, verify findings match expected.
**Validate:** `go test ./tests/ -v` -- all tests pass

### Phase 5: Ship

#### Task 5.1: README + push
**Files:** `README.md` (create)
**Do:** README with: problem statement (AI writes code but misses edge cases), demo output (terminal screenshot), install (`go install`), usage examples, what it catches (categories + AI-specific patterns), CI integration example (GitHub Actions), comparison with CodeRabbit/Diffray (zero-config, no account, BYO key), cost (~$0.01-0.05 per review). Push to GitHub as jtsilverman/probe.
**Validate:** `gh repo view jtsilverman/probe` returns repo info

## The One Hard Thing

**Getting Claude to produce consistent, parseable structured reviews.** The challenge is that code review is inherently subjective, and Claude needs to:
1. Return valid JSON every time (not markdown mixed with JSON)
2. Reference exact line numbers from the diff (not hallucinate line numbers)
3. Distinguish real issues from style preferences
4. Catch AI-specific patterns without false-positiving on legitimate defensive code

**Approach:** Use a strict JSON schema in the system prompt with examples. Pre-process the diff to include line numbers in the content. Use Claude's JSON mode if available, otherwise parse with fallback regex extraction. Validate line numbers against the actual diff and discard findings with invalid references.

**Fallback:** If structured JSON output is unreliable, switch to asking Claude for a markdown-formatted review and parse it with regex into findings. Less precise but more robust.

## Risks

- **Medium -- Output consistency.** Claude may produce inconsistent JSON or hallucinate line numbers. Mitigation: strict schema, validation layer, retry once on parse failure.
- **Low -- API cost.** Large diffs could be expensive. Mitigation: cap at 20 files, split large diffs, estimate cost before sending, show cost in output.
- **Low -- Competition.** CodeRabbit and Diffray exist. Mitigation: zero-config and AI-pattern detection are clear differentiators. Single binary distribution is a plus for CI.
- **Low -- Go Anthropic SDK.** Need to verify go-anthropic package is maintained. Fallback: raw HTTP calls to the API.
