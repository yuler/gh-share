package cmd

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/spf13/cobra"
	"github.com/yuler/gh-share/embedded"
	"github.com/yuler/gh-share/internal/config"
	"github.com/yuler/gh-share/internal/git"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the share repository on GitHub and local machine",
	Long:  "Creates <owner>/share as a public repo, enables GitHub Pages, sets up local clone and starter assets.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Resolve(repoFlag, dirFlag)
		if err != nil {
			return err
		}

		fmt.Printf("🔍 Checking GitHub repository %s...\n", cfg.FullRepo)

		// 1. Check if repo exists
		client, err := api.DefaultRESTClient()
		if err != nil {
			return err
		}

		var repoResp struct {
			HTMLURL string `json:"html_url"`
		}
		repoExists := true
		if err := client.Get(fmt.Sprintf("repos/%s", cfg.FullRepo), &repoResp); err != nil {
			repoExists = false
		}

		if !repoExists {
			fmt.Printf("📦 Creating public repository %s on GitHub...\n", cfg.FullRepo)
			createCmd := exec.Command("gh", "repo", "create", cfg.FullRepo, "--public")
			var errBuf bytes.Buffer
			createCmd.Stderr = &errBuf
			if err := createCmd.Run(); err != nil {
				return fmt.Errorf("failed to create repo: %s (%w)", errBuf.String(), err)
			}
		}

		// 2. Ensure GitHub Pages is enabled
		fmt.Printf("🌐 Ensuring GitHub Pages is enabled for %s...\n", cfg.FullRepo)
		
		// Check if Pages is already enabled using API
		pagesCheckCmd := exec.Command("gh", "api", fmt.Sprintf("/repos/%s/pages", cfg.FullRepo))
		if err := pagesCheckCmd.Run(); err != nil {
			// Pages not enabled, enable it with correct JSON format
			enableCmd := exec.Command("sh", "-c", fmt.Sprintf("echo '{\"source\":{\"branch\":\"main\",\"path\":\"/\"}}' | gh api -X POST /repos/%s/pages --input -", cfg.FullRepo))
			if err := enableCmd.Run(); err != nil {
				fmt.Printf("⚠️  Warning: Failed to enable GitHub Pages: %v\n", err)
				fmt.Printf("   You may need to manually enable GitHub Pages in repository settings\n")
			} else {
				fmt.Printf("✓ GitHub Pages enabled successfully\n")
			}
		} else {
			fmt.Printf("✓ GitHub Pages already enabled\n")
		}

		// 3. Ensure local clone
		fmt.Printf("📂 Ensuring local repository at %s...\n", cfg.LocalDir)
		if err := git.EnsureRepo(cfg.LocalDir, cfg.FullRepo); err != nil {
			return err
		}

		// 4. Ensure starter files exist
		cssFile := filepath.Join(cfg.LocalDir, "assets", "share.css")
		if _, err := os.Stat(cssFile); os.IsNotExist(err) {
			fmt.Println("✨ Bootstrapping templates and assets into local repo...")
			if err := extractEmbedded(cfg.LocalDir); err != nil {
				return err
			}
			_ = git.AddAndCommit(cfg.LocalDir, ".", "🎉 init: starter templates and assets")
			_ = git.Push(cfg.LocalDir)
		}

		fmt.Printf("\n✅ Ready! Share site initialized.\n")
		fmt.Printf("   Local repo : %s\n", cfg.LocalDir)
		fmt.Printf("   Pages URL  : %s\n", cfg.BasePagesURL)
		fmt.Printf("   Command    : gh share post <file|text>\n\n")
		return nil
	},
}

func extractEmbedded(destDir string) error {
	return fs.WalkDir(embedded.Content, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Skip skill folder when bootstrapping the share site
		if len(path) >= 6 && path[:6] == "skill/" {
			return nil
		}

		data, err := fs.ReadFile(embedded.Content, path)
		if err != nil {
			return err
		}

		outPath := filepath.Join(destDir, path)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}

		return os.WriteFile(outPath, data, 0644)
	})
}

func init() {
	rootCmd.AddCommand(initCmd)
}
