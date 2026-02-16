package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zhrkvl/ralph-go/internal/prd"
)

// CheckAndArchive implements the branch-change-detection logic from ralph.sh.
// If the branch in prd.json differs from .last-branch, archives the previous run.
func CheckAndArchive(ralphDir string, p *prd.PRD) (bool, error) {
	lastBranchFile := filepath.Join(ralphDir, ".last-branch")

	lastBranch, err := os.ReadFile(lastBranchFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("reading .last-branch: %w", err)
	}

	lastBranchStr := strings.TrimSpace(string(lastBranch))
	if lastBranchStr == "" || p.BranchName == "" || lastBranchStr == p.BranchName {
		return false, nil
	}

	// Archive previous run
	date := time.Now().Format("2006-01-02")
	folderName := strings.TrimPrefix(lastBranchStr, "ralph/")
	archiveFolder := filepath.Join(ralphDir, "archive", date+"-"+folderName)

	if err := os.MkdirAll(archiveFolder, 0755); err != nil {
		return false, fmt.Errorf("creating archive folder: %w", err)
	}

	// Copy prd.json
	prdPath := filepath.Join(ralphDir, "prd.json")
	if data, err := os.ReadFile(prdPath); err == nil {
		os.WriteFile(filepath.Join(archiveFolder, "prd.json"), data, 0644)
	}

	// Copy progress.txt
	progressPath := filepath.Join(ralphDir, "progress.txt")
	if data, err := os.ReadFile(progressPath); err == nil {
		os.WriteFile(filepath.Join(archiveFolder, "progress.txt"), data, 0644)
	}

	// Reset progress.txt
	resetProgressFile(ralphDir)

	return true, nil
}

// UpdateLastBranch writes the current branch to .last-branch.
func UpdateLastBranch(ralphDir string, branch string) error {
	if branch == "" {
		return nil
	}
	return os.WriteFile(filepath.Join(ralphDir, ".last-branch"), []byte(branch+"\n"), 0644)
}

// InitProgressFile creates progress.txt if it doesn't exist.
func InitProgressFile(ralphDir string) error {
	path := filepath.Join(ralphDir, "progress.txt")
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}
	return resetProgressFile(ralphDir)
}

func resetProgressFile(ralphDir string) error {
	path := filepath.Join(ralphDir, "progress.txt")
	content := fmt.Sprintf("# Ralph Progress Log\nStarted: %s\n---\n", time.Now().Format(time.UnixDate))
	return os.WriteFile(path, []byte(content), 0644)
}

// ArchiveEntry represents an archived session found in the archive/ directory.
type ArchiveEntry struct {
	Path       string
	Date       string
	BranchName string
	HasPRD     bool
	HasProgress bool
}

// ListArchives returns all archived sessions sorted by date (newest first).
func ListArchives(ralphDir string) ([]ArchiveEntry, error) {
	archiveDir := filepath.Join(ralphDir, "archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var archives []ArchiveEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// Expected format: YYYY-MM-DD-branchName
		var date, branch string
		if len(name) >= 10 {
			date = name[:10]
			if len(name) > 11 {
				branch = name[11:]
			}
		} else {
			date = name
		}

		path := filepath.Join(archiveDir, name)
		_, prdErr := os.Stat(filepath.Join(path, "prd.json"))
		_, progErr := os.Stat(filepath.Join(path, "progress.txt"))

		archives = append(archives, ArchiveEntry{
			Path:        path,
			Date:        date,
			BranchName:  branch,
			HasPRD:      prdErr == nil,
			HasProgress: progErr == nil,
		})
	}

	// Sort newest first
	for i, j := 0, len(archives)-1; i < j; i, j = i+1, j-1 {
		archives[i], archives[j] = archives[j], archives[i]
	}

	return archives, nil
}
