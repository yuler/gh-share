package cmd

import (
	"bytes"
	"fmt"
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
	editOpen    bool
	editCopy    bool
	editDraft   bool
	editMessage string
)

var editCmd = &cobra.Command{
	Use:   "edit <slug>",
	Short: "Open an existing share in your text editor",
	Long: `Opens s/<slug>/index.html in your default text editor ($VISUAL, $EDITOR, nano, vim, vi, code).
When the editor closes, if changes were made, automatically commits and pushes to GitHub.

Examples:
  gh share edit 2026-11-04-worker-pool
  gh share edit worker-pool -m "Fix typo in lede"
  gh share edit worker-pool --draft`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Resolve(repoFlag, dirFlag)
		if err != nil {
			return err
		}

		if err := git.EnsureRepo(cfg.LocalDir, cfg.FullRepo); err != nil {
			return err
		}
		if !editDraft {
			_ = git.Pull(cfg.LocalDir)
		}

		slug, err := pages.FindSlug(cfg.LocalDir, args[0])
		if err != nil {
			return err
		}

		targetFile := filepath.Join(cfg.LocalDir, "s", slug, "index.html")
		initialBytes, err := os.ReadFile(targetFile)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", targetFile, err)
		}

		editorBin, editorArgs, err := getEditor()
		if err != nil {
			return err
		}

		fmt.Printf("✏️ Opening %s in %s...\n", targetFile, filepath.Base(editorBin))
		editExec := exec.Command(editorBin, append(editorArgs, targetFile)...)
		editExec.Stdin = os.Stdin
		editExec.Stdout = os.Stdout
		editExec.Stderr = os.Stderr
		if err := editExec.Run(); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		newBytes, err := os.ReadFile(targetFile)
		if err != nil {
			return fmt.Errorf("failed to read %s after editing: %w", targetFile, err)
		}

		if bytes.Equal(initialBytes, newBytes) {
			fmt.Println("No changes made.")
			return nil
		}

		meta := pages.ParsePost(string(newBytes))
		title := meta.Title
		if title == "" {
			title = slug
		}

		if !editDraft {
			relPath := filepath.Join("s", slug)
			commitMsg := editMessage
			if commitMsg == "" {
				commitMsg = fmt.Sprintf("📝 share: edit %s", title)
			}
			if err := git.AddAndCommit(cfg.LocalDir, relPath, commitMsg); err != nil {
				return err
			}
			if err := git.Push(cfg.LocalDir); err != nil {
				return err
			}
		}

		shareURL := fmt.Sprintf("%ss/%s/", strings.TrimRight(cfg.BasePagesURL, "/")+"/", slug)
		if editDraft {
			fmt.Printf("\n✨ Share updated locally (draft mode, not pushed)!\n")
		} else {
			fmt.Printf("\n✨ Share edited successfully!\n")
		}
		fmt.Printf("🔗 URL   : %s\n", shareURL)
		fmt.Printf("📁 Slug  : %s\n", slug)
		fmt.Printf("📄 Title : %s\n\n", title)

		if editCopy {
			if err := clipboard.WriteAll(shareURL); err == nil {
				fmt.Println("📋 URL copied to clipboard.")
			}
		}

		if editOpen {
			openBrowser(shareURL)
		}

		return nil
	},
}

func getEditor() (string, []string, error) {
	editorEnv := os.Getenv("VISUAL")
	if editorEnv == "" {
		editorEnv = os.Getenv("EDITOR")
	}

	if editorEnv != "" {
		parts := strings.Fields(editorEnv)
		if len(parts) > 0 {
			bin, err := exec.LookPath(parts[0])
			if err == nil {
				return bin, parts[1:], nil
			}
		}
	}

	candidates := []string{"editor", "nano", "vim", "vi", "code"}
	for _, c := range candidates {
		bin, err := exec.LookPath(c)
		if err == nil {
			var args []string
			if c == "code" {
				args = []string{"--wait"}
			}
			return bin, args, nil
		}
	}

	return "", nil, fmt.Errorf("no editor found. Please set $EDITOR or $VISUAL environment variable")
}

func init() {
	editCmd.Flags().BoolVarP(&editOpen, "open", "o", false, "Open URL in browser after saving")
	editCmd.Flags().BoolVarP(&editCopy, "copy", "c", false, "Copy URL to clipboard after saving")
	editCmd.Flags().BoolVar(&editDraft, "draft", false, "Edit local file without committing or pushing")
	editCmd.Flags().StringVarP(&editMessage, "message", "m", "", "Custom commit message")

	rootCmd.AddCommand(editCmd)
}
