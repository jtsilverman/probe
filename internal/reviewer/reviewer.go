package reviewer

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/jtsilverman/probe/internal/diff"
)

const (
	maxReviewTokens = 150000 // ~600K chars, well within Claude's context
	sonnetInputCost  = 3.0   // $/MTok
	sonnetOutputCost = 15.0  // $/MTok
)

type ReviewConfig struct {
	APIKey   string
	Model    string
	MaxFiles int
	UseCLI   bool // Use `claude --print` instead of API
}

func RunReview(files []diff.FileDiff, cfg ReviewConfig) (*Review, error) {
	if len(files) == 0 {
		return &Review{
			Findings: []Finding{},
			Summary:  ComputeSummary(nil),
			Model:    cfg.Model,
		}, nil
	}

	// Limit files
	if len(files) > cfg.MaxFiles {
		files = files[:cfg.MaxFiles]
	}

	// Split if too large
	batches := SplitByFile(files, maxReviewTokens)

	var allFindings []Finding
	var totalTokens TokenUsage

	for _, batch := range batches {
		var findings []Finding
		var tokens TokenUsage
		var err error

		if cfg.UseCLI {
			findings, tokens, err = reviewBatchCLI(batch, cfg.Model)
		} else {
			client := anthropic.NewClient(option.WithAPIKey(cfg.APIKey))
			findings, tokens, err = reviewBatch(client, batch, cfg.Model)
		}
		if err != nil {
			return nil, err
		}
		allFindings = append(allFindings, findings...)
		totalTokens.Input += tokens.Input
		totalTokens.Output += tokens.Output
	}

	totalTokens.Cost = float64(totalTokens.Input)/1_000_000*sonnetInputCost +
		float64(totalTokens.Output)/1_000_000*sonnetOutputCost

	model := cfg.Model
	if cfg.UseCLI {
		model = "claude-cli (" + cfg.Model + ")"
		totalTokens.Cost = 0 // Uses subscription, not API billing
	}

	return &Review{
		Findings: allFindings,
		Summary:  ComputeSummary(allFindings),
		Model:    model,
		Tokens:   totalTokens,
	}, nil
}

func reviewBatch(client anthropic.Client, files []diff.FileDiff, model string) ([]Finding, TokenUsage, error) {
	prompt := BuildReviewPrompt(files)
	systemPrompt := GetSystemPrompt()

	resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("claude API error: %w", err)
	}

	tokens := TokenUsage{
		Input:  int(resp.Usage.InputTokens),
		Output: int(resp.Usage.OutputTokens),
	}

	// Extract text from response
	var responseText string
	for _, block := range resp.Content {
		if block.Type == "text" {
			responseText += block.Text
		}
	}

	findings, err := parseFindings(responseText)
	if err != nil {
		return nil, tokens, fmt.Errorf("failed to parse review: %w", err)
	}

	return findings, tokens, nil
}

func reviewBatchCLI(files []diff.FileDiff, model string) ([]Finding, TokenUsage, error) {
	prompt := BuildReviewPrompt(files)
	systemPrompt := GetSystemPrompt()

	fullPrompt := systemPrompt + "\n\n" + prompt

	// Pipe prompt via stdin to claude --print
	cmd := exec.Command("claude", "--print", "--model", model)
	cmd.Stdin = strings.NewReader(fullPrompt)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("claude CLI error: %w\n%s\n(is claude installed and authenticated?)", err, string(out))
	}

	responseText := string(out)
	findings, err := parseFindings(responseText)
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to parse CLI review: %w", err)
	}

	return findings, TokenUsage{}, nil
}

func parseFindings(raw string) ([]Finding, error) {
	raw = strings.TrimSpace(raw)

	// Try to extract JSON from the response
	// Claude might wrap it in markdown code blocks
	if idx := strings.Index(raw, "```json"); idx >= 0 {
		raw = raw[idx+7:]
		if end := strings.Index(raw, "```"); end >= 0 {
			raw = raw[:end]
		}
	} else if idx := strings.Index(raw, "```"); idx >= 0 {
		raw = raw[idx+3:]
		if end := strings.Index(raw, "```"); end >= 0 {
			raw = raw[:end]
		}
	}
	raw = strings.TrimSpace(raw)

	// Try parsing as {"findings": [...]}
	var result struct {
		Findings []Finding `json:"findings"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err == nil {
		return result.Findings, nil
	}

	// Try parsing as bare array [...]
	var findings []Finding
	if err := json.Unmarshal([]byte(raw), &findings); err == nil {
		return findings, nil
	}

	// Last resort: try to find JSON object in the text
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		substr := raw[start : end+1]
		if err := json.Unmarshal([]byte(substr), &result); err == nil {
			return result.Findings, nil
		}
	}

	return nil, fmt.Errorf("could not parse Claude response as JSON: %.200s", raw)
}
