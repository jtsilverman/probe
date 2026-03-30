package reviewer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	cacheTTL = 24 * time.Hour
	cacheDir = ".cache/probe"
)

type cacheEntry struct {
	Findings  []Finding `json:"findings"`
	Timestamp int64     `json:"timestamp"`
}

func cacheKey(diffContent, model, systemPrompt string) string {
	h := sha256.New()
	h.Write([]byte(diffContent))
	h.Write([]byte(model))
	h.Write([]byte(systemPrompt))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func cacheDir2() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, cacheDir)
}

// GetCached checks for a valid cached review result.
func GetCached(diffContent, model, prompt string) ([]Finding, bool) {
	dir := cacheDir2()
	if dir == "" {
		return nil, false
	}

	key := cacheKey(diffContent, model, prompt)
	path := filepath.Join(dir, key+".json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}

	// Check TTL
	if time.Since(time.Unix(entry.Timestamp, 0)) > cacheTTL {
		os.Remove(path)
		return nil, false
	}

	return entry.Findings, true
}

// SetCached stores a review result in the cache.
func SetCached(diffContent, model, prompt string, findings []Finding) {
	dir := cacheDir2()
	if dir == "" {
		return
	}

	os.MkdirAll(dir, 0755)

	key := cacheKey(diffContent, model, prompt)
	entry := cacheEntry{
		Findings:  findings,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	os.WriteFile(filepath.Join(dir, key+".json"), data, 0644)
}
