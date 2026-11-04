# gh-share

[![Release](https://github.com/yuler/gh-share/actions/workflows/release.yml/badge.svg)](https://github.com/yuler/gh-share/actions)

GitHub CLI extension (`gh share`) for zero-friction static sharing via GitHub Pages (`<user>/share`).

Share AI conversations, design ideas, code snippets, notes, or media with a single command. No backend, no heavy build pipelines, just semantic HTML + CSS custom properties served directly via GitHub Pages.

## Installation

Install as a GitHub CLI extension:

```bash
gh extension install yuler/gh-share
```

## Quick Start

### 1. Initialize your share repository

Sets up `<your-username>/share` on GitHub, enables GitHub Pages, and clones starter assets to `~/Projects/share`:

```bash
gh share init
```

### 2. Install the Agent Skill globally

Install the embedded `share` skill across agent environments (`~/.agents/skills/share`, `~/.gemini/config/skills/share`, `~/.claude/skills/share`):

```bash
gh share skill install
```

### 3. Share anything

- **Share a code snippet:**
  ```bash
  gh share post worker.go --title "Worker Pool Pattern" --lede "Concurrency demo"
  ```
- **Share a note or idea:**
  ```bash
  echo "# My Idea\n\nContent..." | gh share post --title "Product Vision" --type note
  ```
- **Share an AI conversation or raw HTML:**
  ```bash
  gh share post chat.html --type chat --title "Architecture Refactor Discussion"
  ```
- **Share an image or video:**
  ```bash
  gh share post screenshot.png --type media --title "New UI Layout"
  ```
- **Copy link or open immediately in browser:**
  ```bash
  gh share post snippet.ts --title "Debounce helper" -c -o
  ```

### 4. Manage and update your shares

```bash
# List all published shares
gh share list

# Output as JSON
gh share list --json

# Open a share in your default browser
gh share open 2026-11-04-worker-pool-pattern

# Update content or metadata of an existing share (preserves untouched metadata)
gh share update 2026-11-04-worker-pool-pattern new_worker.go
gh share update 2026-11-04-worker-pool-pattern --title "New Title" --theme dark

# Append content to an existing share
gh share append 2026-11-04-worker-pool-pattern translation.md
gh share append 2026-11-04-worker-pool-pattern "## Benchmark Results\n..."

# Open a share in your text editor ($VISUAL or $EDITOR) and auto-push on save
gh share edit 2026-11-04-worker-pool-pattern

# Remove a share
gh share rm 2026-11-04-worker-pool-pattern
```

## Conventions

- Default target repository: `<gh-username>/share` (can be overridden with `-R <owner>/<repo>` or `GH_SHARE_REPO`).
- Output URL structure: `https://<owner>.github.io/share/s/<slug>/`.
- Slug format: `YYYY-MM-DD-<kebab-title>`.
- Built-in themes: `paper` (serif), `dark`, `terminal` (monospace green), or default (auto light/dark).

## License

MIT
