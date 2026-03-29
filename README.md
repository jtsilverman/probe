# probe

Zero-config CLI that reviews code using Claude and catches issues linters miss -- especially AI-generated code anti-patterns.

![probe demo](assets/demo.png)

## The Problem

AI writes code fast but misses edge cases. Linters catch syntax errors but not logic bugs, security holes, or patterns that are uniquely AI-generated: hallucinated function calls, overly broad try/catch, copy-paste inconsistencies, undefined variables that look intentional.

CodeRabbit and Diffray exist but require accounts, YAML config, and platform integration. Probe is zero-config: point it at code, get a review. One binary, BYO API key.

## Install

```bash
go install github.com/jtsilverman/probe@latest
```

Or clone and build:
```bash
git clone https://github.com/jtsilverman/probe.git
cd probe && go build -o probe .
```

## Usage

```bash
export ANTHROPIC_API_KEY=sk-ant-...

# Review staged changes (most common flow)
probe

# Review your branch vs main
probe --branch main

# Review a specific file
probe --file src/api.py

# Pipe anything in
cat sketch.py | probe --stdin

# JSON output for CI
probe --branch main --json

# Only show critical/warning (skip info)
probe --severity warning

# Only show security issues
probe --category security
```

### CI Integration (GitHub Actions)

```yaml
- name: AI Code Review
  run: |
    go install github.com/jtsilverman/probe@latest
    probe --branch main --severity warning
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
```

Exit code 1 on critical findings, so `probe || exit 1` works as a CI gate.

## What It Catches

| Category | Examples |
|----------|----------|
| **security** | SQL injection, hardcoded secrets, XSS, insecure deserialization |
| **bug** | Undefined variables, off-by-one errors, null dereferences, race conditions |
| **ai-pattern** | Hallucinated imports, bare except clauses, unused variables, inconsistent error handling |
| **performance** | N+1 queries, unnecessary loops, blocking I/O in async contexts |
| **style** | Inconsistent naming, dead code, overly complex conditionals |

The **ai-pattern** category is unique to Probe. It catches patterns that are specifically common in AI-generated code:
- Imports of functions/modules that don't exist (hallucinated APIs)
- Overly defensive error handling that swallows errors silently
- Variables assigned but never used (that look like they should be)
- Copy-pasted blocks with subtle differences between them

## Output Formats

```bash
probe                    # Colored terminal (default)
probe --json             # JSON (for CI/tooling)
probe --markdown         # GitHub-compatible markdown
```

## Cost

Each review costs ~$0.01-0.05 depending on diff size (Claude Sonnet pricing: $3/$15 per MTok input/output). Token usage and cost are shown in every review output.

## The Hard Part

Getting Claude to produce consistent, parseable JSON reviews with accurate line numbers. The prompt engineering required:
1. Strict JSON schema in the system prompt with explicit field definitions
2. Line numbers embedded directly in the diff content (L1, L2, etc.) so Claude can reference them
3. Validation layer that discards findings with invalid line references
4. Fallback JSON parsing (handles markdown code blocks, bare arrays, nested objects)

## Tech Stack

- **Go** for single-binary distribution and fast CI cold starts
- **Anthropic SDK** (official Go SDK) for Claude API
- **cobra** for CLI framework
- No external dependencies beyond the Claude API

## Options

```
probe [review] [flags]

Flags:
      --branch string     Review diff against branch (e.g., main)
      --file string       Review a specific file
      --stdin             Read diff from stdin
      --json              Output as JSON
      --markdown          Output as GitHub markdown
      --fix               Include fix suggestions
      --severity string   Minimum severity: info, warning, critical (default "info")
      --category string   Filter: bug,security,performance,style,ai-pattern
      --model string      Claude model (default "claude-sonnet-4-20250514")
      --max-files int     Max files to review (default 20)

Environment:
  ANTHROPIC_API_KEY       Required. Get one at console.anthropic.com
```

## License

MIT
