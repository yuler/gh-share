---
name: share
description: Publish content as a public web page at yuler.github.io/share and return the link — an AI/agent conversation, an idea, a code snippet, images or video. Use when the user asks to share something or wants a public link for it.
---

# share

Turns content into a public static web page on GitHub Pages (`yuler/share`) and returns its public URL.

## Slug rules (required)

- Slug / folder / URL path must be **ASCII English kebab-case only**: `YYYY-MM-DD-<english-words>`.
- **Never** put Chinese or other non-ASCII characters in `--slug` or the directory name — titles and body may be Chinese; the slug must stay English so URLs are easy to search, copy, and share.
- When the title is not English (or is ambiguous), **always** pass an explicit English `--slug`.
- Prefer short concrete English: `google-colab-guide`, `worker-pool-pattern`, not pinyin dumps or literal translations of every word.

Examples:

```bash
# Title can be Chinese; slug must be English
gh share post note.md --title "Google Colab 调研与使用指南" --slug 2026-11-05-google-colab-guide

# English title can rely on auto slug, or still set --slug
gh share post snippet.ts --title "Debounce helper" --slug 2026-11-05-debounce-helper
```

## Recommended: `gh share post`

The fastest way to share content is using the `gh share` CLI:

- **Share a file or snippet**:
  ```bash
  gh share post path/to/snippet.ts --title "Debounce helper" --lede "Short explanation" --slug 2026-11-05-debounce-helper
  ```
- **Share an AI conversation or note**:
  ```bash
  gh share post chat.html --type chat --title "Architecture discussion" --slug 2026-11-05-architecture-discussion
  ```
- **Share from standard input**:
  ```bash
  echo "# My Idea\n\nContent..." | gh share post --title "Product Idea" --type note --slug 2026-11-05-product-idea
  ```
- **Update an existing share**:
  ```bash
  gh share update <slug> new_version.md
  gh share update <slug> --title "Updated Title"
  ```
- **Append content to a share**:
  ```bash
  gh share append <slug> translation.md
  echo "## Update note" | gh share append <slug>
  ```
- **Edit an existing share in editor**:
  ```bash
  gh share edit <slug>
  ```
- **Rename a slug** (keeps old links working via an HTML redirect stub; not HTTP 302):
  ```bash
  gh share mv 2026-11-05-google-colab-调研与使用指南 2026-11-05-google-colab-guide
  gh share mv <old-slug> <new-slug> --no-redirect
  ```
- **List existing shares**:
  ```bash
  gh share list
  ```

## Direct repo workflow

If you need full control over custom HTML or multiple bundled assets (e.g. images next to `index.html`):
1. **Repo**: Work in `~/Projects/share`. If missing: `gh repo clone yuler/share ~/Projects/share`.
2. **Conventions**: Slug `YYYY-MM-DD-<english-kebab>` (ASCII only). Themes: `paper`, `dark`, `terminal`. See `~/Projects/share/AGENTS.md` if present.
3. **Redact**: Strip tokens, keys, passwords, private emails, phone numbers, and local paths (`/home/...`).
4. **Push**:
   ```bash
   git add s/<slug> && git commit -m "📝 share: <title>" && git push
   ```

## Completion

Reply with the public URL (`https://yuler.github.io/share/s/<slug>/`) and mention that GitHub Pages updates within ~1 minute.
