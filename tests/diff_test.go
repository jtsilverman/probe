package tests

import (
	"strings"
	"testing"

	"github.com/jtsilverman/probe/internal/diff"
)

const sampleDiff = `diff --git a/src/api.py b/src/api.py
index abc1234..def5678 100644
--- a/src/api.py
+++ b/src/api.py
@@ -1,5 +1,7 @@
 import flask
+import os

 app = flask.Flask(__name__)
+app.config["SECRET"] = os.getenv("SECRET_KEY")

 @app.route("/")
@@ -10,3 +12,6 @@
 def index():
     return "hello"
+
+def get_user(id):
+    return db.query(f"SELECT * FROM users WHERE id = {id}")
`

const newFileDiff = `diff --git a/utils.go b/utils.go
new file mode 100644
--- /dev/null
+++ b/utils.go
@@ -0,0 +1,5 @@
+package main
+
+func helper() string {
+    return "hello"
+}
`

func TestParseUnifiedDiff(t *testing.T) {
	files := diff.ParseUnifiedDiff(sampleDiff)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	f := files[0]
	if f.Path != "src/api.py" {
		t.Errorf("expected path src/api.py, got %s", f.Path)
	}
	if f.Language != "Python" {
		t.Errorf("expected Python, got %s", f.Language)
	}
	if f.IsNew {
		t.Error("should not be marked as new file")
	}
	if len(f.Hunks) != 2 {
		t.Errorf("expected 2 hunks, got %d", len(f.Hunks))
	}

	// Check first hunk
	h1 := f.Hunks[0]
	addCount := 0
	for _, l := range h1.Lines {
		if l.Type == "add" {
			addCount++
		}
	}
	if addCount != 2 {
		t.Errorf("expected 2 additions in first hunk, got %d", addCount)
	}
}

func TestParseNewFile(t *testing.T) {
	files := diff.ParseUnifiedDiff(newFileDiff)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if !files[0].IsNew {
		t.Error("should be marked as new file")
	}
	if files[0].Path != "utils.go" {
		t.Errorf("expected utils.go, got %s", files[0].Path)
	}
	if files[0].Language != "Go" {
		t.Errorf("expected Go, got %s", files[0].Language)
	}
}

func TestParseMultipleFiles(t *testing.T) {
	combined := sampleDiff + "\n" + newFileDiff
	files := diff.ParseUnifiedDiff(combined)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestFormatForReview(t *testing.T) {
	files := diff.ParseUnifiedDiff(sampleDiff)
	formatted := diff.FormatForReview(files)

	if !strings.Contains(formatted, "=== File: src/api.py (Python) ===") {
		t.Error("missing file header")
	}
	if !strings.Contains(formatted, "import os") {
		t.Error("missing added line content")
	}
	if !strings.Contains(formatted, "L") {
		t.Error("missing line numbers")
	}
}

func TestDetectLanguages(t *testing.T) {
	cases := map[string]string{
		"app.py": "Python", "index.js": "JavaScript", "main.go": "Go",
		"lib.rs": "Rust", "App.tsx": "TypeScript", "style.css": "CSS",
	}

	for path, expectedLang := range cases {
		d := "diff --git a/" + path + " b/" + path + "\n--- a/" + path + "\n+++ b/" + path + "\n@@ -1,1 +1,1 @@\n+test\n"
		files := diff.ParseUnifiedDiff(d)
		if len(files) != 1 {
			t.Errorf("%s: expected 1 file, got %d", path, len(files))
			continue
		}
		if files[0].Language != expectedLang {
			t.Errorf("%s: expected %s, got %s", path, expectedLang, files[0].Language)
		}
	}
}

func TestEmptyDiff(t *testing.T) {
	files := diff.ParseUnifiedDiff("")
	if len(files) != 0 {
		t.Errorf("expected 0 files for empty diff, got %d", len(files))
	}
}
