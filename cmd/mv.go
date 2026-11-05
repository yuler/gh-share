package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yuler/gh-share/internal/config"
	"github.com/yuler/gh-share/internal/git"
	"github.com/yuler/gh-share/internal/pages"
)

var mvNoRedirect bool

var asciiSlugReg = regexp.MustCompile(`^[a-zA-Z0-9]+(?:-[a-zA-Z0-9]+)*$`)

var mvCmd = &cobra.Command{
	Use:     "mv <old-slug> <new-slug>",
	Aliases: []string{"rename"},
	Short:   "Rename a share slug, leaving an HTML redirect at the old path",
	Long: `Renames s/<old-slug>/ to s/<new-slug>/, commits, and pushes.

By default a minimal HTML redirect stub is left at the old path so previous
links keep working on GitHub Pages (client-side redirect; not HTTP 301/302).

  gh share mv 2026-11-05-google-colab-调研 2026-11-05-google-colab-guide
  gh share mv old-slug new-slug --no-redirect`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Resolve(repoFlag, dirFlag)
		if err != nil {
			return err
		}

		if err := git.EnsureRepo(cfg.LocalDir, cfg.FullRepo); err != nil {
			return err
		}
		_ = git.Pull(cfg.LocalDir)

		oldSlug, err := pages.FindSlugDir(cfg.LocalDir, args[0])
		if err != nil {
			return err
		}

		newSlug := pages.CleanSlug(args[1])
		if newSlug == "" {
			return fmt.Errorf("new slug cannot be empty")
		}
		if !asciiSlugReg.MatchString(newSlug) {
			return fmt.Errorf("new slug must be ASCII kebab-case (letters, digits, hyphens): %q", newSlug)
		}
		if oldSlug == newSlug {
			return fmt.Errorf("old and new slug are the same: %s", oldSlug)
		}

		oldDir := filepath.Join(cfg.LocalDir, "s", oldSlug)
		newDir := filepath.Join(cfg.LocalDir, "s", newSlug)
		if _, err := os.Stat(oldDir); os.IsNotExist(err) {
			return fmt.Errorf("share '%s' not found at %s", oldSlug, oldDir)
		}
		if _, err := os.Stat(newDir); err == nil {
			return fmt.Errorf("destination slug already exists: %s", newSlug)
		}

		// Refuse to rename a redirect stub (use rm on the stub, or mv the target)
		if htmlContent, err := os.ReadFile(filepath.Join(oldDir, "index.html")); err == nil {
			if pages.IsRedirect(string(htmlContent)) {
				target := pages.RedirectTarget(string(htmlContent))
				if target != "" {
					return fmt.Errorf("'%s' is a redirect to '%s'; rename the target share instead", oldSlug, target)
				}
				return fmt.Errorf("'%s' is a redirect stub; remove it with 'gh share rm' or rename the target share", oldSlug)
			}
		}

		oldRel := filepath.Join("s", oldSlug)
		newRel := filepath.Join("s", newSlug)

		fmt.Printf("🔀 Moving share '%s' → '%s'...\n", oldSlug, newSlug)
		if err := git.Move(cfg.LocalDir, oldRel, newRel); err != nil {
			return err
		}

		paths := []string{newRel}
		leftRedirect := !mvNoRedirect
		if leftRedirect {
			if err := os.MkdirAll(oldDir, 0755); err != nil {
				return fmt.Errorf("failed to create redirect dir %s: %w", oldDir, err)
			}
			newURL := fmt.Sprintf("%ss/%s/", strings.TrimRight(cfg.BasePagesURL, "/")+"/", newSlug)
			stub := pages.BuildRedirectHTML(newSlug, newURL)
			stubPath := filepath.Join(oldDir, "index.html")
			if err := os.WriteFile(stubPath, []byte(stub), 0644); err != nil {
				return fmt.Errorf("failed to write redirect stub: %w", err)
			}
			paths = append(paths, oldRel)
		}

		commitMsg := fmt.Sprintf("🔀 mv share: %s -> %s", oldSlug, newSlug)
		if err := git.AddPathsAndCommit(cfg.LocalDir, paths, commitMsg); err != nil {
			return err
		}
		if err := git.Push(cfg.LocalDir); err != nil {
			return err
		}

		oldURL := fmt.Sprintf("%ss/%s/", strings.TrimRight(cfg.BasePagesURL, "/")+"/", oldSlug)
		newURL := fmt.Sprintf("%ss/%s/", strings.TrimRight(cfg.BasePagesURL, "/")+"/", newSlug)

		fmt.Printf("\n✅ Share renamed successfully!\n")
		fmt.Printf("🔗 New URL : %s\n", newURL)
		fmt.Printf("📁 Slug    : %s → %s\n", oldSlug, newSlug)
		if leftRedirect {
			fmt.Printf("↪️  Redirect: %s → %s (HTML stub)\n\n", oldURL, newURL)
		} else {
			fmt.Printf("↪️  Redirect: none (--no-redirect); old URL will 404\n\n")
		}
		return nil
	},
}

func init() {
	mvCmd.Flags().BoolVar(&mvNoRedirect, "no-redirect", false, "Do not leave an HTML redirect stub at the old path")
	rootCmd.AddCommand(mvCmd)
}
