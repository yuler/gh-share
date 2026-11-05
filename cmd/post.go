package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
	"github.com/yuler/gh-share/internal/config"
	"github.com/yuler/gh-share/internal/git"
	"github.com/yuler/gh-share/internal/pages"
)

var (
	postTitle    string
	postSlug     string
	postType     string
	postTheme    string
	postLede     string
	postTags     []string
	postLang     string
	postOpen     bool
	postCopy     bool
	postDraft    bool
)

var postCmd = &cobra.Command{
	Use:     "post [file or text]",
	Aliases: []string{"create", "publish"},
	Short:   "Publish a new share (note, chat, code, or media)",
	Long: `Creates a new share under s/<YYYY-MM-DD-slug>/, commits, pushes, and returns the public URL.

Examples:
  gh share post snippet.go --title "Go routine worker"
  gh share post chat.html --type chat --title "Architecture discussion"
  gh share post --title "Quick thought" "Here is a note..."
  echo "# Markdown note" | gh share post --title "Product Spec"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Resolve(repoFlag, dirFlag)
		if err != nil {
			return err
		}

		var content string
		var sourceFile string
		var detectedLang string

		if len(args) > 0 {
			input := args[0]
			if info, err := os.Stat(input); err == nil && !info.IsDir() {
				sourceFile = input
				data, err := os.ReadFile(input)
				if err != nil {
					return fmt.Errorf("failed to read file %s: %w", input, err)
				}
				content = string(data)

				ext := strings.ToLower(filepath.Ext(input))
				if postTitle == "" {
					base := filepath.Base(input)
					postTitle = strings.TrimSuffix(base, ext)
				}

				if postType == "" {
					switch ext {
					case ".html", ".htm":
						postType = "raw"
					case ".md", ".markdown", ".txt":
						postType = "note"
					case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".mp4":
						postType = "media"
					default:
						postType = "code"
						detectedLang = strings.TrimPrefix(ext, ".")
					}
				}
			} else {
				content = input
			}
		} else {
			// Read from stdin
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				bytes, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read from stdin: %w", err)
				}
				content = string(bytes)
			}
		}

		if strings.TrimSpace(content) == "" && postType != "media" {
			return fmt.Errorf("no content provided. Pass a file path, text string, or pipe via stdin")
		}

		if postTitle == "" {
			// Try first line of content
			firstLine := strings.TrimSpace(strings.Split(content, "\n")[0])
			firstLine = strings.TrimPrefix(firstLine, "# ")
			if len(firstLine) > 0 && len(firstLine) < 60 {
				postTitle = firstLine
			} else {
				postTitle = "Share " + postType
			}
		}

		if postType == "" {
			postType = "note"
		}

		if postLang == "" && detectedLang != "" {
			postLang = detectedLang
		}

		// Ensure local repo exists and is up to date
		if err := git.EnsureRepo(cfg.LocalDir, cfg.FullRepo); err != nil {
			return err
		}
		if !postDraft {
			_ = git.Pull(cfg.LocalDir)
		}

		// Determine slug
		slug := postSlug
		if slug == "" {
			slug = pages.GenerateSlug(postTitle)
		}

		targetDir := filepath.Join(cfg.LocalDir, "s", slug)
		if _, err := os.Stat(targetDir); err == nil && postSlug == "" {
			// Suffix with incremental counter if slug collision
			for i := 2; i < 100; i++ {
				altSlug := fmt.Sprintf("%s-%d", slug, i)
				altDir := filepath.Join(cfg.LocalDir, "s", altSlug)
				if _, err := os.Stat(altDir); os.IsNotExist(err) {
					slug = altSlug
					targetDir = altDir
					break
				}
			}
		}

		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
		}

		// If media file, copy it into targetDir
		if postType == "media" && sourceFile != "" {
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
				content = fmt.Sprintf(`<figure class="wide"><img src="%s" alt="%s"></figure>`, relName, postTitle)
			}
		}

		opts := pages.CreateOptions{
			Title:    postTitle,
			Slug:     slug,
			Type:     postType,
			Theme:    postTheme,
			Lede:     postLede,
			Tags:     postTags,
			CodeLang: postLang,
			CodeFile: filepath.Base(sourceFile),
		}

		htmlContent, err := pages.BuildHTML(content, opts, cfg.BasePagesURL)
		if err != nil {
			return fmt.Errorf("failed to build HTML: %w", err)
		}

		indexPath := filepath.Join(targetDir, "index.html")
		if err := os.WriteFile(indexPath, []byte(htmlContent), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", indexPath, err)
		}

		shareURL := fmt.Sprintf("%ss/%s/", strings.TrimRight(cfg.BasePagesURL, "/")+"/", slug)

		if !postDraft {
			relPath := filepath.Join("s", slug)
			commitMsg := fmt.Sprintf("📝 share: %s", postTitle)
			if err := git.AddAndCommit(cfg.LocalDir, relPath, commitMsg); err != nil {
				return err
			}
			if err := git.Push(cfg.LocalDir); err != nil {
				return err
			}
		}

		fmt.Printf("\n✨ Share published successfully!\n")
		fmt.Printf("🔗 URL   : %s\n", shareURL)
		fmt.Printf("📁 Slug  : %s\n", slug)
		fmt.Printf("📄 Type  : %s\n\n", postType)

		if postCopy {
			if err := clipboard.WriteAll(shareURL); err == nil {
				fmt.Println("📋 URL copied to clipboard.")
			}
		}

		if postOpen {
			openBrowser(shareURL)
		}

		return nil
	},
}

func openBrowser(url string) {
	_ = exec.Command("xdg-open", url).Start()
}

func init() {
	postCmd.Flags().StringVarP(&postTitle, "title", "t", "", "Title of the share")
	postCmd.Flags().StringVarP(&postSlug, "slug", "s", "", "Custom slug (ASCII English kebab; default: YYYY-MM-DD-<title>)")
	postCmd.Flags().StringVar(&postType, "type", "", "Type: note, chat, code, media, or raw")
	postCmd.Flags().StringVar(&postTheme, "theme", "", "Theme: paper, dark, terminal")
	postCmd.Flags().StringVar(&postLede, "lede", "", "Subtitle / summary lede")
	postCmd.Flags().StringSliceVar(&postTags, "tag", nil, "Tags for the share (comma-separated or repeated)")
	postCmd.Flags().StringVar(&postLang, "code-lang", "", "Language for code block syntax highlighting")
	postCmd.Flags().BoolVarP(&postOpen, "open", "o", false, "Open URL in browser after publish")
	postCmd.Flags().BoolVarP(&postCopy, "copy", "c", false, "Copy URL to clipboard after publish")
	postCmd.Flags().BoolVar(&postDraft, "draft", false, "Create local file without committing or pushing")

	rootCmd.AddCommand(postCmd)
}
