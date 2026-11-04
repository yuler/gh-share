package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yuler/gh-share/internal/config"
	"github.com/yuler/gh-share/internal/git"
	"github.com/yuler/gh-share/internal/pages"
)

var rmCmd = &cobra.Command{
	Use:     "rm <slug>",
	Aliases: []string{"delete"},
	Short:   "Delete a share from the repository",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Resolve(repoFlag, dirFlag)
		if err != nil {
			return err
		}

		slug, err := pages.FindSlug(cfg.LocalDir, args[0])
		if err != nil {
			return err
		}

		targetDir := filepath.Join(cfg.LocalDir, "s", slug)
		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			return fmt.Errorf("share '%s' not found at %s", slug, targetDir)
		}

		fmt.Printf("🗑️ Removing share '%s'...\n", slug)
		relPath := filepath.Join("s", slug)
		commitMsg := fmt.Sprintf("🗑️ rm share: %s", slug)
		if err := git.RemoveAndCommit(cfg.LocalDir, relPath, commitMsg); err != nil {
			return err
		}

		fmt.Printf("✅ Removed and pushed.\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
