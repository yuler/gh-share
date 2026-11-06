package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToKebabCaseASCIIOnly(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Debounce helper", "debounce-helper"},
		{"Google Colab 调研与使用指南", "google-colab"},
		{"纯中文标题", ""},
		{"RTX PRO 6000!", "rtx-pro-6000"},
	}
	for _, tc := range tests {
		actual := ToKebabCase(tc.input)
		if actual != tc.expected {
			t.Errorf("ToKebabCase(%q) = %q, expected %q", tc.input, actual, tc.expected)
		}
	}
}

func TestCleanSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2026-11-04-hello-world", "2026-11-04-hello-world"},
		{"s/2026-11-04-hello-world", "2026-11-04-hello-world"},
		{"/s/2026-11-04-hello-world/", "2026-11-04-hello-world"},
		{"https://yuler.github.io/share/s/2026-11-04-hello-world/", "2026-11-04-hello-world"},
		{"https://yuler.github.io/share/s/2026-11-04-hello-world?foo=bar#section", "2026-11-04-hello-world"},
		{"  s/my-slug/  ", "my-slug"},
	}

	for _, tc := range tests {
		actual := CleanSlug(tc.input)
		if actual != tc.expected {
			t.Errorf("CleanSlug(%q) = %q, expected %q", tc.input, actual, tc.expected)
		}
	}
}

func TestFindSlug(t *testing.T) {
	tempDir := t.TempDir()
	sDir := filepath.Join(tempDir, "s")

	shares := []string{
		"2026-11-04-worker-pool-pattern",
		"2026-11-04-hello-share",
		"2026-11-05-browser-automation",
	}

	for _, slug := range shares {
		dir := filepath.Join(sDir, slug)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// 1. Exact match
	slug, err := FindSlug(tempDir, "2026-11-04-hello-share")
	if err != nil || slug != "2026-11-04-hello-share" {
		t.Errorf("expected 2026-11-04-hello-share, got %s (err: %v)", slug, err)
	}

	// 2. Stripped date match
	slug, err = FindSlug(tempDir, "hello-share")
	if err != nil || slug != "2026-11-04-hello-share" {
		t.Errorf("expected 2026-11-04-hello-share, got %s (err: %v)", slug, err)
	}

	// 3. Substring match
	slug, err = FindSlug(tempDir, "browser")
	if err != nil || slug != "2026-11-05-browser-automation" {
		t.Errorf("expected 2026-11-05-browser-automation, got %s (err: %v)", slug, err)
	}

	// 4. URL input
	slug, err = FindSlug(tempDir, "https://user.github.io/share/s/worker-pool-pattern/")
	if err != nil || slug != "2026-11-04-worker-pool-pattern" {
		t.Errorf("expected 2026-11-04-worker-pool-pattern, got %s (err: %v)", slug, err)
	}

	// 5. Ambiguous match
	_, err = FindSlug(tempDir, "2026-11-04")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected ambiguous error, got %v", err)
	}

	// 6. Not found
	_, err = FindSlug(tempDir, "nonexistent-post")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found error, got %v", err)
	}
}

func TestParsePostAndBuildHTML(t *testing.T) {
	sampleHTML := `<!doctype html>
<html lang="zh-CN" data-theme="paper">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="robots" content="noindex, nofollow">
  <title>Worker Pool Pattern &amp; Concurrency</title>
  <meta name="description" content="A demonstration of worker pools in Go.">
  <meta property="og:title" content="Worker Pool Pattern &amp; Concurrency">
  <meta property="og:description" content="A demonstration of worker pools in Go.">
  <link rel="stylesheet" href="../../assets/share.css">
  <script src="../../assets/share.js" defer></script>
</head>
<body>
  <main class="page">
    <header class="meta">
      <h1>Worker Pool Pattern &amp; Concurrency</h1>
      <p><time datetime="2026-11-04">2026-11-04</time> <span class="tag">golang</span> <span class="tag">concurrency</span> </p>
      <p class="lede">A demonstration of worker pools in Go.</p>
    </header>

    <h2>Introduction</h2>
    <p>Worker pools are useful for limiting concurrency.</p>

    <footer>shared via <a href="https://example.com/share/">gh-share</a></footer>
  </main>
</body>
</html>`

	meta := ParsePost(sampleHTML)
	if meta.Title != "Worker Pool Pattern & Concurrency" {
		t.Errorf("expected title 'Worker Pool Pattern & Concurrency', got %q", meta.Title)
	}
	if meta.Date != "2026-11-04" {
		t.Errorf("expected date '2026-11-04', got %q", meta.Date)
	}
	if meta.Theme != "paper" {
		t.Errorf("expected theme 'paper', got %q", meta.Theme)
	}
	if meta.Lede != "A demonstration of worker pools in Go." {
		t.Errorf("expected lede 'A demonstration of worker pools in Go.', got %q", meta.Lede)
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "golang" || meta.Tags[1] != "concurrency" {
		t.Errorf("expected tags [golang concurrency], got %v", meta.Tags)
	}
	if meta.FooterURL != "https://example.com/share/" {
		t.Errorf("expected footer URL 'https://example.com/share/', got %q", meta.FooterURL)
	}
	if meta.RawHTML {
		t.Errorf("expected RawHTML to be false")
	}
	expectedBody := `<h2>Introduction</h2>
    <p>Worker pools are useful for limiting concurrency.</p>`
	if strings.TrimSpace(meta.BodyHTML) != strings.TrimSpace(expectedBody) {
		t.Errorf("expected body %q, got %q", expectedBody, meta.BodyHTML)
	}

	// Test rebuild preserving metadata
	opts := CreateOptions{
		Title: meta.Title,
		Date:  meta.Date,
		Theme: meta.Theme,
		Lede:  meta.Lede,
		Tags:  meta.Tags,
		Type:  "html",
	}
	rebuilt, err := BuildHTML(meta.BodyHTML, opts, meta.FooterURL)
	if err != nil {
		t.Fatal(err)
	}
	rebuiltMeta := ParsePost(rebuilt)
	if rebuiltMeta.Title != meta.Title {
		t.Errorf("rebuilt title mismatch: %q vs %q", rebuiltMeta.Title, meta.Title)
	}
	if rebuiltMeta.Theme != meta.Theme {
		t.Errorf("rebuilt theme mismatch: %q vs %q", rebuiltMeta.Theme, meta.Theme)
	}
	if rebuiltMeta.Date != meta.Date {
		t.Errorf("rebuilt date mismatch: %q vs %q", rebuiltMeta.Date, meta.Date)
	}
	if rebuiltMeta.Lede != meta.Lede {
		t.Errorf("rebuilt lede mismatch: %q vs %q", rebuiltMeta.Lede, meta.Lede)
	}
	if len(rebuiltMeta.Tags) != len(meta.Tags) {
		t.Errorf("rebuilt tags mismatch: %v vs %v", rebuiltMeta.Tags, meta.Tags)
	}
}

func TestMarkdownExtensions(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "raw html",
			input:    "This is <b>bold</b> and <kbd>Ctrl</kbd>",
			contains: []string{"<b>bold</b>", "<kbd>Ctrl</kbd>"},
		},
		{
			name:     "cjk bold brackets",
			input:    "它就像一个**「云端版的 Python 笔记本」**——你不需要",
			contains: []string{"<strong>「云端版的 Python 笔记本」</strong>"},
		},
		{
			name:     "cjk book title",
			input:    "前缀**《书名》**后缀",
			contains: []string{"<strong>《书名》</strong>"},
		},
		{
			name:     "cjk strikethrough",
			input:    "这是~~「删除」~~内容",
			contains: []string{"<del>「删除」</del>"},
		},
		{
			name:     "cjk line break",
			input:    "第一行中文\n第二行中文",
			contains: []string{"第一行中文第二行中文"},
		},
		{
			name:     "table",
			input:    "| A | B |\n|---|---|\n| 1 | 2 |",
			contains: []string{"<table>", "<th>A</th>", "<td>1</td>"},
		},
		{
			name:     "autolink",
			input:    "Visit https://google.com directly",
			contains: []string{`<a href="https://google.com">https://google.com</a>`},
		},
		{
			name:     "tasklist",
			input:    "- [x] Done\n- [ ] Todo",
			contains: []string{`type="checkbox"`},
		},
		{
			name:     "footnote",
			input:    "Here is a note[^1].\n\n[^1]: note body",
			contains: []string{`class="footnote-ref"`, `class="footnotes"`},
		},
	}
	for _, tc := range cases {
		out, err := RenderBodyHTML(tc.input, "note", "", "")
		if err != nil {
			t.Errorf("%s failed: %v", tc.name, err)
			continue
		}
		for _, exp := range tc.contains {
			if !strings.Contains(out, exp) {
				t.Errorf("[%s] expected output to contain %q, got:\n%s", tc.name, exp, out)
			}
		}
	}
}

func TestAppendBodyHTML(t *testing.T) {
	// Standard markdown notes
	existing := "<p>First paragraph.</p>"
	appended := "<p>Second paragraph.</p>"
	result := AppendBodyHTML(existing, appended, "note")
	expected := "<p>First paragraph.</p>\n\n<p>Second paragraph.</p>"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	// Chat posts
	chatExisting := `<section class="chat">
      <div class="msg" data-role="user"><p>Hi</p></div>
    </section>`
	chatAppended := `<div class="msg" data-role="assistant"><p>Hello there</p></div>`
	chatResult := AppendBodyHTML(chatExisting, chatAppended, "chat")
	if !strings.Contains(chatResult, `<div class="msg" data-role="assistant"><p>Hello there</p></div>`) {
		t.Errorf("expected assistant msg in chat, got %s", chatResult)
	}
	if !strings.HasSuffix(strings.TrimSpace(chatResult), "</section>") {
		t.Errorf("expected chatResult to end with </section>, got %s", chatResult)
	}
}

func TestReplaceTitle(t *testing.T) {
	doc := `<html><head><title>Old Title</title></head><body></body></html>`
	updated := ReplaceTitle(doc, "New Title & More")
	if !strings.Contains(updated, "<title>New Title &amp; More</title>") {
		t.Errorf("expected escaped new title, got %s", updated)
	}
}

func TestBuildRedirectHTML(t *testing.T) {
	abs := "https://yuler.github.io/share/s/2026-11-05-google-colab-guide/"
	htmlOut := BuildRedirectHTML("2026-11-05-google-colab-guide", abs)

	if !IsRedirect(htmlOut) {
		t.Fatal("expected IsRedirect true")
	}
	if !strings.Contains(htmlOut, `data-share-type="redirect"`) {
		t.Fatal("missing data-share-type")
	}
	if !strings.Contains(htmlOut, `url=../2026-11-05-google-colab-guide/`) {
		t.Fatalf("missing meta refresh relative url, got:\n%s", htmlOut)
	}
	if !strings.Contains(htmlOut, abs) {
		t.Fatalf("missing absolute url, got:\n%s", htmlOut)
	}
	if target := RedirectTarget(htmlOut); target != "2026-11-05-google-colab-guide" {
		t.Fatalf("RedirectTarget = %q, want 2026-11-05-google-colab-guide", target)
	}
}

func TestIsRedirectFalseForNormalShare(t *testing.T) {
	doc := `<html lang="zh-CN" data-theme="paper"><head><title>Hi</title></head></html>`
	if IsRedirect(doc) {
		t.Fatal("normal share should not be a redirect")
	}
}

func TestListSharesSkipsRedirects(t *testing.T) {
	tempDir := t.TempDir()
	sDir := filepath.Join(tempDir, "s")

	realSlug := "2026-11-05-google-colab-guide"
	oldSlug := "2026-11-05-google-colab-old"
	realDir := filepath.Join(sDir, realSlug)
	oldDir := filepath.Join(sDir, oldSlug)
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatal(err)
	}
	realHTML := `<html><head><title>Colab Guide</title></head><body><header class="meta"><h1>Colab Guide</h1></header></body></html>`
	abs := "https://example.com/share/s/" + realSlug + "/"
	if err := os.WriteFile(filepath.Join(realDir, "index.html"), []byte(realHTML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "index.html"), []byte(BuildRedirectHTML(realSlug, abs)), 0644); err != nil {
		t.Fatal(err)
	}

	items, err := ListShares(tempDir, "https://example.com/share/")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 share (redirect skipped), got %d: %+v", len(items), items)
	}
	if items[0].Slug != realSlug {
		t.Fatalf("expected %s, got %s", realSlug, items[0].Slug)
	}
}

func TestFindSlugFollowsRedirect(t *testing.T) {
	tempDir := t.TempDir()
	sDir := filepath.Join(tempDir, "s")

	realSlug := "2026-11-05-google-colab-guide"
	oldSlug := "2026-11-05-google-colab-调研"
	for _, slug := range []string{realSlug, oldSlug} {
		if err := os.MkdirAll(filepath.Join(sDir, slug), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(sDir, realSlug, "index.html"), []byte("<html><title>Real</title></html>"), 0644); err != nil {
		t.Fatal(err)
	}
	abs := "https://example.com/share/s/" + realSlug + "/"
	if err := os.WriteFile(filepath.Join(sDir, oldSlug, "index.html"), []byte(BuildRedirectHTML(realSlug, abs)), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := FindSlug(tempDir, oldSlug)
	if err != nil {
		t.Fatal(err)
	}
	if got != realSlug {
		t.Fatalf("FindSlug followed redirect: got %q, want %q", got, realSlug)
	}

	dir, err := FindSlugDir(tempDir, oldSlug)
	if err != nil {
		t.Fatal(err)
	}
	if dir != oldSlug {
		t.Fatalf("FindSlugDir should not follow: got %q, want %q", dir, oldSlug)
	}
}

func TestFindSlugBrokenRedirect(t *testing.T) {
	tempDir := t.TempDir()
	sDir := filepath.Join(tempDir, "s")
	oldSlug := "2026-11-05-missing-target"
	if err := os.MkdirAll(filepath.Join(sDir, oldSlug), 0755); err != nil {
		t.Fatal(err)
	}
	stub := BuildRedirectHTML("2026-11-05-gone", "https://example.com/share/s/2026-11-05-gone/")
	if err := os.WriteFile(filepath.Join(sDir, oldSlug, "index.html"), []byte(stub), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := FindSlug(tempDir, oldSlug)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing target error, got %v", err)
	}
}
