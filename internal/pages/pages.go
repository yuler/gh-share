package pages

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tats-u/goldmark-cjk-friendly/v2"
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	goldmarkHtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type gfmCJKStrikethroughParser struct {
	inner parser.InlineParser
}

func (p *gfmCJKStrikethroughParser) Trigger() []byte {
	return []byte{'~'}
}

func (p *gfmCJKStrikethroughParser) Parse(parent gast.Node, block text.Reader, pc parser.Context) gast.Node {
	line, _ := block.PeekLine()
	// GFM strikethrough strictly requires double tildes '~~'.
	// Single tildes (e.g. ~168h, ~7 days) or more than two tildes must NOT be parsed as strikethrough.
	if len(line) < 2 || line[0] != '~' || line[1] != '~' || (len(line) > 2 && line[2] == '~') {
		return nil
	}
	return p.inner.Parse(parent, block, pc)
}

type cjkStrikethroughExtension struct{}

func (e *cjkStrikethroughExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&gfmCJKStrikethroughParser{inner: cjkfriendly.NewCJKFriendlyStrikethroughParser()}, 499),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(extension.NewStrikethroughHTMLRenderer(), 500),
	))
}

// markdown converts CommonMark + GFM (tables, strikethrough, autolink, task lists)
// with CJK-friendly emphasis & strikethrough, East Asian line breaks, footnotes, and raw HTML.
var markdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		cjkfriendly.CJKFriendlyEmphasis,
		&cjkStrikethroughExtension{},
		extension.NewCJK(extension.WithEastAsianLineBreaks(), extension.WithEscapedSpace()),
		extension.Footnote,
	),
	goldmark.WithRendererOptions(
		goldmarkHtml.WithUnsafe(),
	),
)

type ShareItem struct {
	Slug  string    `json:"slug"`
	Date  string    `json:"date"`
	Title string    `json:"title"`
	Lede  string    `json:"lede,omitempty"`
	URL   string    `json:"url"`
	Path  string    `json:"path"`
	MTime time.Time `json:"mtime"`
}

type CreateOptions struct {
	Title    string
	Slug     string
	Date     string // YYYY-MM-DD
	Type     string // note, chat, code, media, raw, html, raw_body
	Theme    string // paper, dark, terminal, or empty
	Lede     string
	Tags     []string
	CodeLang string
	CodeFile string
}

type PostMetadata struct {
	Slug      string   `json:"slug"`
	Title     string   `json:"title"`
	Date      string   `json:"date"`
	Theme     string   `json:"theme,omitempty"`
	Lede      string   `json:"lede,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	BodyHTML  string   `json:"body_html,omitempty"`
	RawHTML   bool     `json:"raw_html"`
	FooterURL string   `json:"footer_url,omitempty"`
}

var (
	titleReg           = regexp.MustCompile(`(?i)<title>(.*?)</title>`)
	h1Reg              = regexp.MustCompile(`(?i)<h1[^>]*>(.*?)</h1>`)
	ledeReg            = regexp.MustCompile(`(?i)<p class="lede">(.*?)</p>`)
	themeReg           = regexp.MustCompile(`(?i)<html[^>]*data-theme="([^"]*)"`)
	timeReg            = regexp.MustCompile(`(?i)<time[^>]*datetime="([^"]*)"`)
	tagReg             = regexp.MustCompile(`(?i)<span class="tag">(.*?)</span>`)
	footerReg          = regexp.MustCompile(`(?i)<footer[^>]*>[\s\S]*?<a href="([^"]*)".*?</footer>`)
	headerReg          = regexp.MustCompile(`(?i)<header class="meta">[\s\S]*?</header>`)
	redirectTypeReg    = regexp.MustCompile(`(?i)data-share-type\s*=\s*["']redirect["']`)
	redirectCanonical  = regexp.MustCompile(`(?i)<link[^>]*rel=["']canonical["'][^>]*href=["']([^"']+)["']`)
	redirectRefreshURL = regexp.MustCompile(`(?i)<meta[^>]*http-equiv=["']refresh["'][^>]*content=["'][^"']*url=([^"'\s>]+)`)
	redirectAnchorHref = regexp.MustCompile(`(?i)<a\s+href=["']([^"']+)["']`)
)

// ReplaceTitle replaces the <title> tag content in an HTML string
func ReplaceTitle(htmlContent string, newTitle string) string {
	escaped := html.EscapeString(newTitle)
	if titleReg.MatchString(htmlContent) {
		return titleReg.ReplaceAllString(htmlContent, fmt.Sprintf("<title>%s</title>", escaped))
	}
	return htmlContent
}

// CleanSlug extracts the slug from URLs, paths, or raw slug strings
func CleanSlug(input string) string {
	s := strings.TrimSpace(input)
	if idx := strings.Index(s, "/s/"); idx != -1 {
		s = s[idx+3:]
	} else if strings.HasPrefix(s, "s/") {
		s = strings.TrimPrefix(s, "s/")
	}
	s = strings.Trim(s, "/")
	if idx := strings.Index(s, "?"); idx != -1 {
		s = s[:idx]
	}
	if idx := strings.Index(s, "#"); idx != -1 {
		s = s[:idx]
	}
	return s
}

// BuildRedirectHTML builds a minimal HTML redirect stub for GitHub Pages.
// Relative target is ../<newSlug>/; absoluteURL is used for canonical and body link.
func BuildRedirectHTML(newSlug, absoluteURL string) string {
	rel := "../" + newSlug + "/"
	escRel := html.EscapeString(rel)
	escAbs := html.EscapeString(absoluteURL)
	return fmt.Sprintf(`<!doctype html>
<html lang="en" data-share-type="redirect">
<head>
  <meta charset="utf-8">
  <meta name="robots" content="noindex">
  <meta http-equiv="refresh" content="0; url=%s">
  <link rel="canonical" href="%s">
  <title>Moved</title>
  <script>location.replace(%q);</script>
</head>
<body>
  <p>Moved to <a href="%s">%s</a>.</p>
</body>
</html>
`, escRel, escAbs, rel, escRel, escAbs)
}

// IsRedirect reports whether htmlContent is a share redirect stub.
func IsRedirect(htmlContent string) bool {
	return redirectTypeReg.MatchString(htmlContent)
}

// RedirectTarget extracts the destination slug from a redirect stub.
// Prefers relative ../slug/ paths; falls back to CleanSlug of absolute URLs.
func RedirectTarget(htmlContent string) string {
	var href string
	if m := redirectRefreshURL.FindStringSubmatch(htmlContent); len(m) > 1 {
		href = strings.TrimSpace(m[1])
	} else if m := redirectCanonical.FindStringSubmatch(htmlContent); len(m) > 1 {
		href = strings.TrimSpace(m[1])
	} else if m := redirectAnchorHref.FindStringSubmatch(htmlContent); len(m) > 1 {
		href = strings.TrimSpace(m[1])
	}
	if href == "" {
		return ""
	}
	href = html.UnescapeString(href)
	// Relative form: ../new-slug/ or ./new-slug/
	if strings.HasPrefix(href, "../") || strings.HasPrefix(href, "./") {
		return CleanSlug(filepath.Base(strings.TrimSuffix(href, "/")))
	}
	return CleanSlug(href)
}

// readShareIndex returns index.html contents for slug, or empty if missing.
func readShareIndex(localDir, slug string) (string, error) {
	indexPath := filepath.Join(localDir, "s", slug, "index.html")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// resolveRedirect follows redirect stubs to the final share slug (max 5 hops).
func resolveRedirect(localDir, slug string) (string, error) {
	seen := map[string]bool{}
	current := slug
	for i := 0; i < 5; i++ {
		if seen[current] {
			return "", fmt.Errorf("share '%s' has a redirect cycle", slug)
		}
		seen[current] = true

		htmlContent, err := readShareIndex(localDir, current)
		if err != nil {
			return current, nil
		}
		if !IsRedirect(htmlContent) {
			return current, nil
		}
		target := RedirectTarget(htmlContent)
		if target == "" {
			return "", fmt.Errorf("share '%s' is a redirect but has no target", current)
		}
		targetIndex := filepath.Join(localDir, "s", target, "index.html")
		if _, err := os.Stat(targetIndex); err != nil {
			return "", fmt.Errorf("share '%s' redirects to '%s', but that share was not found", current, target)
		}
		current = target
	}
	return "", fmt.Errorf("share '%s' redirect chain is too long", slug)
}

// FindSlug locates a share directory under localDir/s matching the input slug.
// If the match is a redirect stub, it follows to the target share when present.
func FindSlug(localDir string, input string) (string, error) {
	return findSlug(localDir, input, true)
}

// FindSlugDir locates a share directory without following redirect stubs.
// Used by mv so renaming targets the literal folder.
func FindSlugDir(localDir string, input string) (string, error) {
	return findSlug(localDir, input, false)
}

func findSlug(localDir string, input string, followRedirects bool) (string, error) {
	slug := CleanSlug(input)
	if slug == "" {
		return "", fmt.Errorf("slug cannot be empty")
	}

	sDir := filepath.Join(localDir, "s")
	if _, err := os.Stat(sDir); os.IsNotExist(err) {
		return "", fmt.Errorf("no shares directory found at %s", sDir)
	}

	finish := func(name string) (string, error) {
		if followRedirects {
			return resolveRedirect(localDir, name)
		}
		return name, nil
	}

	// 1. Direct exact match
	exactDir := filepath.Join(sDir, slug)
	exactIndex := filepath.Join(exactDir, "index.html")
	if info, err := os.Stat(exactIndex); err == nil && !info.IsDir() {
		return finish(slug)
	}

	// 2. Scan s/ directory
	entries, err := os.ReadDir(sDir)
	if err != nil {
		return "", fmt.Errorf("failed to read shares directory: %w", err)
	}

	var candidates []string
	slugLower := strings.ToLower(slug)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		indexPath := filepath.Join(sDir, name, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			continue
		}

		nameLower := strings.ToLower(name)
		if nameLower == slugLower {
			return finish(name)
		}

		// Check stripped date (e.g. YYYY-MM-DD-slug -> slug)
		nameWithoutDate := nameLower
		if len(nameLower) >= 11 && nameLower[4] == '-' && nameLower[7] == '-' && nameLower[10] == '-' {
			nameWithoutDate = nameLower[11:]
		}

		if nameWithoutDate == slugLower {
			candidates = append(candidates, name)
			continue
		}

		// Check if substring match
		if strings.Contains(nameLower, slugLower) {
			candidates = append(candidates, name)
		}
	}

	candidates = removeDuplicates(candidates)

	if len(candidates) == 1 {
		return finish(candidates[0])
	}
	if len(candidates) > 1 {
		return "", fmt.Errorf("ambiguous share '%s', multiple matches found: %s (please specify full slug)", input, strings.Join(candidates, ", "))
	}

	return "", fmt.Errorf("share '%s' not found in %s", input, sDir)
}

func removeDuplicates(items []string) []string {
	seen := make(map[string]bool, len(items))
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// ParsePost parses metadata and body content from an existing share HTML document
func ParsePost(htmlContent string) *PostMetadata {
	meta := &PostMetadata{}

	if m := titleReg.FindStringSubmatch(htmlContent); len(m) > 1 {
		meta.Title = html.UnescapeString(strings.TrimSpace(m[1]))
	} else if m := h1Reg.FindStringSubmatch(htmlContent); len(m) > 1 {
		meta.Title = html.UnescapeString(strings.TrimSpace(m[1]))
	}

	if m := themeReg.FindStringSubmatch(htmlContent); len(m) > 1 {
		meta.Theme = strings.TrimSpace(m[1])
	}

	if m := timeReg.FindStringSubmatch(htmlContent); len(m) > 1 {
		meta.Date = strings.TrimSpace(m[1])
	}

	if m := ledeReg.FindStringSubmatch(htmlContent); len(m) > 1 {
		meta.Lede = html.UnescapeString(strings.TrimSpace(m[1]))
	}

	searchHeader := htmlContent
	if hm := headerReg.FindString(htmlContent); hm != "" {
		searchHeader = hm
	}
	tagMatches := tagReg.FindAllStringSubmatch(searchHeader, -1)
	for _, tm := range tagMatches {
		if len(tm) > 1 {
			meta.Tags = append(meta.Tags, html.UnescapeString(strings.TrimSpace(tm[1])))
		}
	}

	if m := footerReg.FindStringSubmatch(htmlContent); len(m) > 1 {
		meta.FooterURL = strings.TrimSpace(m[1])
	}

	lower := strings.ToLower(htmlContent)
	headerEnd := strings.Index(lower, "</header>")
	footerStart := strings.LastIndex(lower, "<footer")

	if headerEnd != -1 && footerStart != -1 && footerStart > headerEnd {
		meta.BodyHTML = strings.TrimSpace(htmlContent[headerEnd+len("</header>") : footerStart])
		meta.RawHTML = false
	} else {
		meta.BodyHTML = htmlContent
		meta.RawHTML = true
	}

	return meta
}

// GenerateSlug generates a slug like 2026-11-04-hello-world from a title
func GenerateSlug(title string) string {
	today := time.Now().Format("2006-01-02")
	kebab := ToKebabCase(title)
	if kebab == "" {
		kebab = "share"
	}
	// Limit to ~5 words / 40 chars
	parts := strings.Split(kebab, "-")
	if len(parts) > 5 {
		parts = parts[:5]
	}
	kebab = strings.Join(parts, "-")
	return fmt.Sprintf("%s-%s", today, kebab)
}

// ToKebabCase keeps ASCII letters/digits only so slugs stay English-searchable URLs.
// Non-ASCII (e.g. Chinese) is dropped; callers should pass an explicit English --slug
// when the title has no Latin words.
func ToKebabCase(s string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	cleaned := reg.ReplaceAllString(s, "-")
	cleaned = strings.ToLower(strings.Trim(cleaned, "-"))
	for strings.Contains(cleaned, "--") {
		cleaned = strings.ReplaceAll(cleaned, "--", "-")
	}
	return cleaned
}

// RenderBodyHTML renders content into an HTML snippet based on postType
func RenderBodyHTML(content string, postType string, codeLang string, codeFile string) (string, error) {
	switch postType {
	case "code":
		lang := codeLang
		if lang == "" {
			lang = "text"
		}
		dataTitle := ""
		if codeFile != "" && codeFile != "." {
			dataTitle = fmt.Sprintf(` data-title="%s"`, html.EscapeString(codeFile))
		}
		escapedCode := html.EscapeString(content)
		return fmt.Sprintf(`<pre%s><code class="language-%s">%s</code></pre>`, dataTitle, html.EscapeString(lang), escapedCode), nil

	case "chat":
		// If content is already chat divs, use it; otherwise wrap lines or parse
		if strings.Contains(content, `class="msg"`) {
			return fmt.Sprintf(`<section class="chat">%s</section>`, content), nil
		}
		return fmt.Sprintf(`<section class="chat"><div class="msg" data-role="user"><p>%s</p></div></section>`, html.EscapeString(content)), nil

	case "media", "raw_body", "html":
		return content, nil

	default: // "note" or markdown
		var buf bytes.Buffer
		if err := markdown.Convert([]byte(content), &buf); err != nil {
			return fmt.Sprintf("<p>%s</p>", html.EscapeString(content)), nil
		}
		return buf.String(), nil
	}
}

// AppendBodyHTML joins existing body HTML and new body HTML cleanly
func AppendBodyHTML(existingBody string, newBody string, newType string) string {
	existingTrimmed := strings.TrimSpace(existingBody)
	newTrimmed := strings.TrimSpace(newBody)

	if existingTrimmed == "" {
		return newTrimmed
	}
	if newTrimmed == "" {
		return existingTrimmed
	}

	// If existing body contains a chat section and new content is a chat message
	if strings.Contains(existingTrimmed, "</section>") && (newType == "chat" || strings.Contains(newTrimmed, `class="msg"`)) {
		lastClose := strings.LastIndex(existingTrimmed, "</section>")
		if lastClose != -1 {
			snippet := newTrimmed
			if strings.HasPrefix(snippet, `<section class="chat">`) && strings.HasSuffix(snippet, `</section>`) {
				snippet = strings.TrimPrefix(snippet, `<section class="chat">`)
				snippet = strings.TrimSuffix(snippet, `</section>`)
				snippet = strings.TrimSpace(snippet)
			}
			return existingTrimmed[:lastClose] + "\n      " + snippet + "\n    " + existingTrimmed[lastClose:]
		}
	}

	return existingTrimmed + "\n\n" + newTrimmed
}

// BuildHTML generates the complete HTML document
func BuildHTML(content string, opts CreateOptions, basePagesURL string) (string, error) {
	trimmed := strings.TrimSpace(content)
	if opts.Type == "raw" || strings.HasPrefix(strings.ToLower(trimmed), "<!doctype html") || strings.HasPrefix(strings.ToLower(trimmed), "<html") {
		return content, nil
	}

	today := opts.Date
	if today == "" {
		today = time.Now().Format("2006-01-02")
	}

	title := opts.Title
	if title == "" {
		title = "Untitled Share"
	}

	themeAttr := ""
	if opts.Theme != "" {
		themeAttr = fmt.Sprintf(` data-theme="%s"`, html.EscapeString(opts.Theme))
	}

	tagsHTML := ""
	for _, tag := range opts.Tags {
		tagsHTML += fmt.Sprintf(`<span class="tag">%s</span> `, html.EscapeString(tag))
	}

	ledeHTML := ""
	if opts.Lede != "" {
		ledeHTML = fmt.Sprintf(`<p class="lede">%s</p>`, html.EscapeString(opts.Lede))
	}

	var bodyHTML string
	if opts.Type == "html" || opts.Type == "raw_body" {
		bodyHTML = content
	} else {
		var err error
		bodyHTML, err = RenderBodyHTML(content, opts.Type, opts.CodeLang, opts.CodeFile)
		if err != nil {
			return "", err
		}
	}

	escapedTitle := html.EscapeString(title)
	escapedDesc := html.EscapeString(opts.Lede)
	if escapedDesc == "" {
		escapedDesc = escapedTitle
	}

	doc := fmt.Sprintf(`<!doctype html>
<html lang="zh-CN"%s>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="robots" content="noindex, nofollow">
  <title>%s</title>
  <meta name="description" content="%s">
  <meta property="og:title" content="%s">
  <meta property="og:description" content="%s">
  <link rel="stylesheet" href="../../assets/share.css">
  <script src="../../assets/share.js" defer></script>
</head>
<body>
  <main class="page">
    <header class="meta">
      <h1>%s</h1>
      <p><time datetime="%s">%s</time> %s</p>
      %s
    </header>

    %s

    <footer>shared via <a href="%s">gh-share</a></footer>
  </main>
</body>
</html>
`, themeAttr, escapedTitle, escapedDesc, escapedTitle, escapedDesc, escapedTitle, today, today, tagsHTML, ledeHTML, bodyHTML, basePagesURL)

	return doc, nil
}

// ListShares scans s/ directory for all shares
func ListShares(localDir string, basePagesURL string) ([]ShareItem, error) {
	sDir := filepath.Join(localDir, "s")
	entries, err := os.ReadDir(sDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var items []ShareItem

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		indexPath := filepath.Join(sDir, slug, "index.html")
		info, err := os.Stat(indexPath)
		if err != nil {
			continue
		}

		item := ShareItem{
			Slug:  slug,
			Path:  indexPath,
			MTime: info.ModTime(),
			URL:   fmt.Sprintf("%ss/%s/", strings.TrimRight(basePagesURL, "/")+"/", slug),
		}

		// Extract date from slug (YYYY-MM-DD)
		if len(slug) >= 10 && slug[4] == '-' && slug[7] == '-' {
			item.Date = slug[:10]
		} else {
			item.Date = info.ModTime().Format("2006-01-02")
		}

		// Parse title from index.html; skip redirect stubs
		htmlBytes, err := os.ReadFile(indexPath)
		if err == nil {
			htmlContent := string(htmlBytes)
			if IsRedirect(htmlContent) {
				continue
			}
			meta := ParsePost(htmlContent)
			if meta.Title != "" {
				item.Title = meta.Title
			}
			if meta.Lede != "" {
				item.Lede = meta.Lede
			}
		}

		if item.Title == "" {
			item.Title = slug
		}

		items = append(items, item)
	}

	// Sort newest first
	sort.Slice(items, func(i, j int) bool {
		if items[i].Date != items[j].Date {
			return items[i].Date > items[j].Date
		}
		return items[i].Slug > items[j].Slug
	})

	return items, nil
}
