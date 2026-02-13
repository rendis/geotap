package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const maxRecent = 10

type RecentEntry struct {
	Path      string    `json:"path"`
	OpenedAt  time.Time `json:"opened_at"`
}

func recentFilePath() string {
	cfg, _ := os.UserConfigDir()
	return filepath.Join(cfg, "geotap", "recent.json")
}

func LoadRecent() []RecentEntry {
	data, err := os.ReadFile(recentFilePath())
	if err != nil {
		return nil
	}
	var entries []RecentEntry
	json.Unmarshal(data, &entries)
	return entries
}

func SaveRecent(dbPath string) {
	abs, err := filepath.Abs(dbPath)
	if err != nil {
		abs = dbPath
	}

	entries := LoadRecent()

	// Remove duplicate
	filtered := make([]RecentEntry, 0, len(entries))
	for _, e := range entries {
		if e.Path != abs {
			filtered = append(filtered, e)
		}
	}

	// Prepend
	filtered = append([]RecentEntry{{Path: abs, OpenedAt: time.Now()}}, filtered...)
	if len(filtered) > maxRecent {
		filtered = filtered[:maxRecent]
	}

	data, _ := json.MarshalIndent(filtered, "", "  ")
	dir := filepath.Dir(recentFilePath())
	os.MkdirAll(dir, 0755)
	os.WriteFile(recentFilePath(), data, 0644)
}
