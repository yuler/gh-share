package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// EnsureRepo ensures that the target repository is cloned locally.
func EnsureRepo(localDir string, fullRepo string) error {
	gitDir := filepath.Join(localDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(localDir), 0755); err != nil {
		return fmt.Errorf("failed to create parent dir: %w", err)
	}

	cmd := exec.Command("gh", "repo", "clone", fullRepo, localDir)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repo %s to %s: %s (%w)", fullRepo, localDir, errBuf.String(), err)
	}
	return nil
}

// Pull pulls the latest commits with rebase
func Pull(localDir string) error {
	cmd := exec.Command("git", "-C", localDir, "pull", "--rebase")
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %s (%w)", errBuf.String(), err)
	}
	return nil
}

// AddAndCommit stages the path and creates a commit
func AddAndCommit(localDir string, relPath string, message string) error {
	return AddPathsAndCommit(localDir, []string{relPath}, message)
}

// AddPathsAndCommit stages one or more paths and creates a commit
func AddPathsAndCommit(localDir string, relPaths []string, message string) error {
	if len(relPaths) == 0 {
		return fmt.Errorf("git add failed: no paths provided")
	}
	args := append([]string{"-C", localDir, "add", "--"}, relPaths...)
	addCmd := exec.Command("git", args...)
	var errBuf bytes.Buffer
	addCmd.Stderr = &errBuf
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %s (%w)", errBuf.String(), err)
	}

	commitCmd := exec.Command("git", "-C", localDir, "commit", "-m", message)
	errBuf.Reset()
	commitCmd.Stderr = &errBuf
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %s (%w)", errBuf.String(), err)
	}
	return nil
}

// Move renames a path inside the repo with git mv.
func Move(localDir string, oldRel, newRel string) error {
	cmd := exec.Command("git", "-C", localDir, "mv", oldRel, newRel)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git mv failed: %s (%w)", errBuf.String(), err)
	}
	return nil
}

// Push pushes commits to remote
func Push(localDir string) error {
	cmd := exec.Command("git", "-C", localDir, "push")
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push failed: %s (%w)", errBuf.String(), err)
	}
	return nil
}

// RemoveAndCommit removes path, commits and pushes
func RemoveAndCommit(localDir string, relPath string, message string) error {
	rmCmd := exec.Command("git", "-C", localDir, "rm", "-rf", relPath)
	var errBuf bytes.Buffer
	rmCmd.Stderr = &errBuf
	if err := rmCmd.Run(); err != nil {
		return fmt.Errorf("git rm failed: %s (%w)", errBuf.String(), err)
	}

	commitCmd := exec.Command("git", "-C", localDir, "commit", "-m", message)
	errBuf.Reset()
	commitCmd.Stderr = &errBuf
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %s (%w)", errBuf.String(), err)
	}

	return Push(localDir)
}
