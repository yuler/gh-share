package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateAndAppendDraft(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	slug := "2026-11-04-test-share"
	shareDir := filepath.Join(tempDir, "s", slug)
	if err := os.MkdirAll(shareDir, 0755); err != nil {
		t.Fatal(err)
	}

	initialHTML := `<!doctype html>
<html lang="zh-CN" data-theme="paper">
<head>
  <meta charset="utf-8">
  <title>Initial Title</title>
  <meta name="description" content="Initial lede">
  <meta property="og:title" content="Initial Title">
  <meta property="og:description" content="Initial lede">
  <link rel="stylesheet" href="../../assets/share.css">
  <script src="../../assets/share.js" defer></script>
</head>
<body>
  <main class="page">
    <header class="meta">
      <h1>Initial Title</h1>
      <p><time datetime="2026-11-04">2026-11-04</time> <span class="tag">initial</span> </p>
      <p class="lede">Initial lede</p>
    </header>

    <p>Initial content</p>

    <footer>shared via <a href="https://example.com/share/">gh-share</a></footer>
  </main>
</body>
</html>`

	indexPath := filepath.Join(shareDir, "index.html")
	if err := os.WriteFile(indexPath, []byte(initialHTML), 0644); err != nil {
		t.Fatal(err)
	}

	// Test 1: Update metadata only
	dirFlag = tempDir
	err := runUpdateShared(updateRunnerOptions{
		slugInput: slug,
		draft:     true,
		title:     "Updated Title",
		titleSet:  true,
		theme:     "dark",
		themeSet:  true,
	})
	if err != nil {
		t.Fatalf("runUpdateShared metadata update failed: %v", err)
	}

	updatedBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	updatedStr := string(updatedBytes)
	if !strings.Contains(updatedStr, "<title>Updated Title</title>") {
		t.Errorf("expected updated title in HTML, got:\n%s", updatedStr)
	}
	if !strings.Contains(updatedStr, `data-theme="dark"`) {
		t.Errorf("expected data-theme='dark', got:\n%s", updatedStr)
	}
	// Initial content should be preserved
	if !strings.Contains(updatedStr, "<p>Initial content</p>") {
		t.Errorf("expected initial content to be preserved, got:\n%s", updatedStr)
	}
	// Initial tag should be preserved
	if !strings.Contains(updatedStr, `<span class="tag">initial</span>`) {
		t.Errorf("expected initial tag to be preserved, got:\n%s", updatedStr)
	}

	// Test 2: Append content
	err = runUpdateShared(updateRunnerOptions{
		slugInput:    slug,
		contentInput: "## Appended Section\n\nMore information.",
		hasContent:   true,
		isAppend:     true,
		draft:        true,
	})
	if err != nil {
		t.Fatalf("runUpdateShared append failed: %v", err)
	}

	appendedBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	appendedStr := string(appendedBytes)
	if !strings.Contains(appendedStr, "<p>Initial content</p>") {
		t.Errorf("expected initial content to be present after append, got:\n%s", appendedStr)
	}
	if !strings.Contains(appendedStr, "<h2>Appended Section</h2>") {
		t.Errorf("expected appended section in HTML, got:\n%s", appendedStr)
	}
	if !strings.Contains(appendedStr, "<p>More information.</p>") {
		t.Errorf("expected appended text in HTML, got:\n%s", appendedStr)
	}
	// Title and theme should be preserved from previous update
	if !strings.Contains(appendedStr, "<title>Updated Title</title>") {
		t.Errorf("expected title to be preserved after append, got:\n%s", appendedStr)
	}
	if !strings.Contains(appendedStr, `data-theme="dark"`) {
		t.Errorf("expected theme to be preserved after append, got:\n%s", appendedStr)
	}

	// Test 3: Replace content
	err = runUpdateShared(updateRunnerOptions{
		slugInput:    slug,
		contentInput: "## Brand New Content Only",
		hasContent:   true,
		isAppend:     false,
		draft:        true,
	})
	if err != nil {
		t.Fatalf("runUpdateShared replace failed: %v", err)
	}

	replacedBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	replacedStr := string(replacedBytes)
	if strings.Contains(replacedStr, "<p>Initial content</p>") {
		t.Errorf("expected initial content to be replaced, got:\n%s", replacedStr)
	}
	if !strings.Contains(replacedStr, "<h2>Brand New Content Only</h2>") {
		t.Errorf("expected new content in HTML, got:\n%s", replacedStr)
	}
}
