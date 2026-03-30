package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents .proberc settings
type Config struct {
	Ignore     []string     `yaml:"ignore"`
	Severity   string       `yaml:"severity"`
	Categories []string     `yaml:"categories"`
	Model      string       `yaml:"model"`
	Rules      []CustomRule `yaml:"rules"`
}

// CustomRule is a user-defined pattern to check for
type CustomRule struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Pattern     string `yaml:"pattern"`
	Severity    string `yaml:"severity"`
	Category    string `yaml:"category"`
}

// Load searches for .proberc starting from the current directory,
// walking up to the git root. Returns an empty config if not found.
func Load() Config {
	dir, err := os.Getwd()
	if err != nil {
		return Config{}
	}

	root := gitRoot()

	for {
		path := filepath.Join(dir, ".proberc")
		data, err := os.ReadFile(path)
		if err == nil {
			var cfg Config
			if yaml.Unmarshal(data, &cfg) == nil {
				return cfg
			}
		}

		// Stop at git root or filesystem root
		if dir == root || dir == "/" || dir == "." {
			break
		}
		dir = filepath.Dir(dir)
	}

	return Config{}
}

// ShouldIgnore checks if a file path matches any ignore pattern
func (c *Config) ShouldIgnore(path string) bool {
	for _, pattern := range c.Ignore {
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
		// Also try matching against the basename
		matched, err = filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
		// Handle ** prefix patterns by checking suffix
		if strings.HasPrefix(pattern, "**/") {
			suffix := strings.TrimPrefix(pattern, "**/")
			suffixMatched, err := filepath.Match(suffix, filepath.Base(path))
			if err == nil && suffixMatched {
				return true
			}
		}
	}
	return false
}

func gitRoot() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
