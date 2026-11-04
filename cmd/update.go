package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
	"github.com/yuler/gh-share/internal/config"
	"github.com/yuler/gh-share/internal/git"
	"github.com/yuler/gh-share/internal/pages"
)

var (
	updateTitle   string
	updateTheme   string
	updateLede    string
	updateTags    []string
	updateType    string
	updateLang    string
	updateAppend  bool
	updateOpen    bool
	updateCopy    bool
	updateDraft   bool
	updateMessage string

	appendType    string
	appendLang    string
	appendOpen    bool
	appendCopy    bool
	appendDraft   bool
	appendMessage string
)

type updateRunnerOptions struct {
	slugInput    string
	contentInput string
	hasContent   bool
	isAppend     bool
	title        string
	theme        string
	lede         string
	tags         []string
	contentType  string
	codeLang     string
	open         bool
	copyURL      bool
	draft        bool
	message      string
	titleSet     bool
	themeSet     bool
	ledeSet      bool
	tagsSet      bool
}

var updateCmd = &cobra.Command{
	Use:     "update <slug> [file or text]",
	Aliases: []string{"modify"},
	Short:   "Update content or metadata of an existing share",
	Long: `Updates an existing share under s/<slug>/, commits, pushes, and returns the public URL.
Preserves existing metadata (title, lede, tags, theme, date) unless explicitly overridden.

Examples:
  gh share update 2026-11-04-worker-pool new_worker.go
  gh share update 2026-11-04-worker-pool "New text content..."
  gh share update 2026-11-04-worker-pool --title "New Title" --theme dark
  gh share update 2026-11-04-worker-pool extra_notes.md --append
  echo "# Updated Spec" | gh share update 2026-11-04-worker-pool`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slugInput := args[0]
		var contentInput string
		hasContent := false
		if len(args) > 1 {
			contentInput = args[1]
			hasContent = true
		}

		opts := updateRunnerOptions{
			slugInput:    slugInput,
			contentInput: contentInput,
			hasContent:   hasContent,
			isAppend:     updateAppend,
			title:        updateTitle,
			theme:        updateTheme,
			lede:         updateLede,
			tags:         updateTags,
			contentType:  updateType,
			codeLang:     updateLang,
			open:         updateOpen,
			copyURL:      updateCopy,
			draft:        updateDraft,
			message:      updateMessage,
			titleSet:     cmd.Flags().Changed("title"),
			themeSet:     cmd.Flags().Changed("theme"),
			ledeSet:      cmd.Flags().Changed("lede"),
			tagsSet:      cmd.Flags().Changed("tag"),
		}

		return runUpdateShared(opts)
	},
}

var appendCmd = &cobra.Command{
	Use:   "append <slug> [file or text]",
	Short: "Append content to an existing share",
	Long: `Appends new content to an existing share under s/<slug>/, commits, pushes, and returns the public URL.

Examples:
  gh share append 2026-11-04-worker-pool translations.md
  gh share append 2026-11-04-worker-pool "## Chinese Translation\n..."
  cat notes.md | gh share append 2026-11-04-worker-pool`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slugInput := args[0]
		var contentInput string
		hasContent := false
		if len(args) > 1 {
			contentInput = args[1]
			hasContent = true
		}

		opts := updateRunnerOptions{
			slugInput:    slugInput,
			contentInput: contentInput,
			hasContent:   hasContent,
			isAppend:     true,
			contentType:  appendType,
			codeLang:     appendLang,
			open:         appendOpen,
			copyURL:      appendCopy,
			draft:        appendDraft,
			message:      appendMessage,
		}

		return runUpdateShared(opts)
	},
}

func runUpdateShared(opts updateRunnerOptions) error {
	cfg, err := config.Resolve(repoFlag, dirFlag)
	if err != nil {
		return err
	}

	if err := git.EnsureRepo(cfg.LocalDir, cfg.FullRepo); err != nil {
		return err
	}
	if !opts.draft {
		_ = git.Pull(cfg.LocalDir)
	}

	slug, err := pages.FindSlug(cfg.LocalDir, opts.slugInput)
	if err != nil {
		return err
	}

	targetDir := filepath.Join(cfg.LocalDir, "s", slug)
	indexPath := filepath.Join(targetDir, "index.html")
	existingData, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("failed to read post at %s: %w", indexPath, err)
	}
	meta := pages.ParsePost(string(existingData))

	var content string
	var sourceFile string
	var detectedLang string

	if opts.hasContent {
		input := opts.contentInput
		if info, err := os.Stat(input); err == nil && !info.IsDir() {
			sourceFile = input
			data, err := os.ReadFile(input)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", input, err)
			}
			content = string(data)

			ext := strings.ToLower(filepath.Ext(input))
			if opts.contentType == "" {
				switch ext {
				case ".html", ".htm":
					opts.contentType = "raw"
				case ".md", ".markdown", ".txt":
					opts.contentType = "note"
				case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".mp4":
					opts.contentType = "media"
				default:
					opts.contentType = "code"
					detectedLang = strings.TrimPrefix(ext, ".")
				}
			}
		} else {
			content = input
		}
	} else {
		// Read from stdin if piped
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			bytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			content = string(bytes)
		}
	}

	hasContentProvided := strings.TrimSpace(content) != "" || opts.contentType == "media"
	hasMetadataChanges := opts.titleSet || opts.themeSet || opts.ledeSet || opts.tagsSet

	if !hasContentProvided && !hasMetadataChanges {
		if opts.isAppend {
			return fmt.Errorf("no content provided to append. Pass a file path, text string, or pipe via stdin")
		}
		return fmt.Errorf("no content or metadata changes provided for share '%s'. Pass a file/text, pipe via stdin, specify flags (--title, --lede, --theme, --tag), or use 'gh share edit %s'", slug, slug)
	}

	finalTitle := meta.Title
	if opts.titleSet {
		finalTitle = opts.title
	} else if finalTitle == "" {
		finalTitle = slug
	}

	finalTheme := meta.Theme
	if opts.themeSet {
		finalTheme = opts.theme
	}

	finalLede := meta.Lede
	if opts.ledeSet {
		finalLede = opts.lede
	}

	finalTags := meta.Tags
	if opts.tagsSet {
		finalTags = opts.tags
	}

	finalDate := meta.Date
	if finalDate == "" {
		if len(slug) >= 10 && slug[4] == '-' && slug[7] == '-' {
			finalDate = slug[:10]
		} else {
			finalDate = time.Now().Format("2006-01-02")
		}
	}

	// If media file, copy it into targetDir
	if opts.contentType == "media" && sourceFile != "" {
		destFile := filepath.Join(targetDir, filepath.Base(sourceFile))
		inputData, err := os.ReadFile(sourceFile)
		if err != nil {
			return err
		}
		if err := os.WriteFile(destFile, inputData, 0644); err != nil {
			return err
		}
		relName := "./" + filepath.Base(sourceFile)
		ext := strings.ToLower(filepath.Ext(sourceFile))
		if ext == ".mp4" || ext == ".webm" {
			content = fmt.Sprintf(`<figure><video controls playsinline src="%s"></video></figure>`, relName)
		} else {
			content = fmt.Sprintf(`<figure class="wide"><img src="%s" alt="%s"></figure>`, relName, finalTitle)
		}
	}

	if opts.codeLang == "" && detectedLang != "" {
		opts.codeLang = detectedLang
	}
	if opts.contentType == "" && hasContentProvided {
		opts.contentType = "note"
	}

	var finalBodyHTML string
	var newHTMLDoc string

	if meta.RawHTML {
		if hasContentProvided {
			if opts.isAppend {
				renderedSnippet, err := pages.RenderBodyHTML(content, opts.contentType, opts.codeLang, filepath.Base(sourceFile))
				if err != nil {
					return err
				}
				bodyLower := strings.ToLower(meta.BodyHTML)
				if idx := strings.LastIndex(bodyLower, "</body>"); idx != -1 {
					newHTMLDoc = meta.BodyHTML[:idx] + "\n" + renderedSnippet + "\n" + meta.BodyHTML[idx:]
				} else {
					newHTMLDoc = meta.BodyHTML + "\n\n" + renderedSnippet
				}
			} else {
				newHTMLDoc = content
			}
		} else {
			newHTMLDoc = meta.BodyHTML
		}

		if opts.titleSet {
			newHTMLDoc = pages.ReplaceTitle(newHTMLDoc, finalTitle)
		}
	} else {
		if hasContentProvided {
			renderedBody, err := pages.RenderBodyHTML(content, opts.contentType, opts.codeLang, filepath.Base(sourceFile))
			if err != nil {
				return fmt.Errorf("failed to render content: %w", err)
			}
			if opts.isAppend {
				finalBodyHTML = pages.AppendBodyHTML(meta.BodyHTML, renderedBody, opts.contentType)
			} else {
				finalBodyHTML = renderedBody
			}
		} else {
			finalBodyHTML = meta.BodyHTML
		}

		createOpts := pages.CreateOptions{
			Title: finalTitle,
			Slug:  slug,
			Date:  finalDate,
			Theme: finalTheme,
			Lede:  finalLede,
			Tags:  finalTags,
			Type:  "html",
		}
		footerURL := meta.FooterURL
		if footerURL == "" {
			footerURL = cfg.BasePagesURL
		}
		var err error
		newHTMLDoc, err = pages.BuildHTML(finalBodyHTML, createOpts, footerURL)
		if err != nil {
			return fmt.Errorf("failed to build HTML: %w", err)
		}
	}

	if err := os.WriteFile(indexPath, []byte(newHTMLDoc), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", indexPath, err)
	}

	if !opts.draft {
		relPath := filepath.Join("s", slug)
		commitMsg := opts.message
		if commitMsg == "" {
			if opts.isAppend {
				commitMsg = fmt.Sprintf("📝 share: append to %s", finalTitle)
			} else {
				commitMsg = fmt.Sprintf("📝 share: update %s", finalTitle)
			}
		}
		if err := git.AddAndCommit(cfg.LocalDir, relPath, commitMsg); err != nil {
			return err
		}
		if err := git.Push(cfg.LocalDir); err != nil {
			return err
		}
	}

	shareURL := fmt.Sprintf("%ss/%s/", strings.TrimRight(cfg.BasePagesURL, "/")+"/", slug)
	action := "updated"
	if opts.isAppend {
		action = "appended to"
	}
	if opts.draft {
		fmt.Printf("\n✨ Share %s locally (draft mode, not pushed)!\n", action)
	} else {
		fmt.Printf("\n✨ Share %s successfully!\n", action)
	}
	fmt.Printf("🔗 URL   : %s\n", shareURL)
	fmt.Printf("📁 Slug  : %s\n", slug)
	fmt.Printf("📄 Title : %s\n\n", finalTitle)

	if opts.copyURL {
		if err := clipboard.WriteAll(shareURL); err == nil {
			fmt.Println("📋 URL copied to clipboard.")
		}
	}

	if opts.open {
		openBrowser(shareURL)
	}

	return nil
}

func init() {
	updateCmd.Flags().StringVarP(&updateTitle, "title", "t", "", "Update title")
	updateCmd.Flags().StringVar(&updateTheme, "theme", "", "Update theme: paper, dark, terminal")
	updateCmd.Flags().StringVar(&updateLede, "lede", "", "Update subtitle / summary lede")
	updateCmd.Flags().StringSliceVar(&updateTags, "tag", nil, "Update tags (comma-separated or repeated)")
	updateCmd.Flags().StringVar(&updateType, "type", "", "Content type: note, chat, code, media, or raw")
	updateCmd.Flags().StringVar(&updateLang, "code-lang", "", "Language for code block syntax highlighting")
	updateCmd.Flags().BoolVarP(&updateAppend, "append", "a", false, "Append content to existing post instead of replacing")
	updateCmd.Flags().BoolVarP(&updateOpen, "open", "o", false, "Open URL in browser after update")
	updateCmd.Flags().BoolVarP(&updateCopy, "copy", "c", false, "Copy URL to clipboard after update")
	updateCmd.Flags().BoolVar(&updateDraft, "draft", false, "Update local file without committing or pushing")
	updateCmd.Flags().StringVarP(&updateMessage, "message", "m", "", "Custom commit message")

	appendCmd.Flags().StringVar(&appendType, "type", "", "Content type: note, chat, code, media, or raw")
	appendCmd.Flags().StringVar(&appendLang, "code-lang", "", "Language for code block syntax highlighting")
	appendCmd.Flags().BoolVarP(&appendOpen, "open", "o", false, "Open URL in browser after append")
	appendCmd.Flags().BoolVarP(&appendCopy, "copy", "c", false, "Copy URL to clipboard after append")
	appendCmd.Flags().BoolVar(&appendDraft, "draft", false, "Append to local file without committing or pushing")
	appendCmd.Flags().StringVarP(&appendMessage, "message", "m", "", "Custom commit message")

	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(appendCmd)
}
