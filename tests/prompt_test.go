package tests

import (
	"strings"
	"testing"

	"github.com/jtsilverman/probe/internal/reviewer"
)

func TestLanguagePromptGo(t *testing.T) {
	prompt := reviewer.GetSystemPromptForLanguage("Go")
	if !strings.Contains(prompt, "Unchecked errors") {
		t.Error("Go prompt should mention unchecked errors")
	}
	if !strings.Contains(prompt, "Goroutine leaks") {
		t.Error("Go prompt should mention goroutine leaks")
	}
}

func TestLanguagePromptPython(t *testing.T) {
	prompt := reviewer.GetSystemPromptForLanguage("Python")
	if !strings.Contains(prompt, "Mutable default") {
		t.Error("Python prompt should mention mutable defaults")
	}
	if !strings.Contains(prompt, "Bare except") {
		t.Error("Python prompt should mention bare except")
	}
}

func TestLanguagePromptJavaScript(t *testing.T) {
	prompt := reviewer.GetSystemPromptForLanguage("JavaScript")
	if !strings.Contains(prompt, "Prototype pollution") {
		t.Error("JS prompt should mention prototype pollution")
	}
}

func TestLanguagePromptRust(t *testing.T) {
	prompt := reviewer.GetSystemPromptForLanguage("Rust")
	if !strings.Contains(prompt, "unwrap()") {
		t.Error("Rust prompt should mention unwrap()")
	}
}

func TestLanguagePromptUnknown(t *testing.T) {
	prompt := reviewer.GetSystemPromptForLanguage("Brainfuck")
	if prompt != reviewer.GetSystemPrompt() {
		t.Error("Unknown language should return base system prompt")
	}
}
