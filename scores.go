package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type scoreStore struct {
	Version   int            `json:"version"`
	HighScore map[string]int `json:"high_scores"`
}

var (
	scoresMu sync.Mutex
	scores   = scoreStore{
		Version:   1,
		HighScore: map[string]int{"easy": 0, "medium": 0, "hard": 0, "custom": 0},
	}
)

func loadScores() error {
	scoresMu.Lock()
	defer scoresMu.Unlock()

	path, err := scoresPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		migrateOldScoresLocked()
		return saveScoresLocked()
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, &scores); err != nil {
		migrateOldScoresLocked()
		return saveScoresLocked()
	}
	if scores.HighScore == nil {
		scores.HighScore = map[string]int{}
	}

	changed := false
	for _, id := range []string{"easy", "medium", "hard", "custom"} {
		if _, ok := scores.HighScore[id]; !ok {
			scores.HighScore[id] = 0
			changed = true
		}
	}
	if scores.Version == 0 {
		scores.Version = 1
		changed = true
	}
	if changed {
		return saveScoresLocked()
	}
	return nil
}

func getHighScore(modeID string) int {
	scoresMu.Lock()
	defer scoresMu.Unlock()
	return scores.HighScore[modeID]
}

func setHighScore(modeID string, score int) error {
	scoresMu.Lock()
	defer scoresMu.Unlock()

	if scores.HighScore == nil {
		scores.HighScore = map[string]int{}
	}
	scores.HighScore[modeID] = score
	return saveScoresLocked()
}

func saveScoresLocked() error {
	path, err := scoresPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	content, err := json.MarshalIndent(scores, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		_ = os.Remove(path)
	}
	return os.Rename(tmp, path)
}

func migrateOldScoresLocked() {
	for _, modeID := range []string{"easy", "medium", "hard", "custom"} {
		score, ok := readOldScore(modeID)
		if !ok {
			continue
		}
		key := modeID
		if modeID == "custom" {
			key = "custom:60"
		}
		if score > scores.HighScore[key] {
			scores.HighScore[key] = score
		}
	}
}

func readOldScore(modeID string) (int, bool) {
	content, err := os.ReadFile(oldScorePath(modeID))
	if err != nil {
		return 0, false
	}
	score, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		return 0, false
	}
	return score, true
}

func scoresPath() (string, error) {
	if override := strings.TrimSpace(os.Getenv("CLICKER_DATA_DIR")); override != "" {
		return filepath.Join(override, "scores.json"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Clicker Game", "scores.json"), nil
}

func oldScorePath(modeID string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(`C:\CLICKER`, strings.ToUpper(modeID)+".txt")
	}
	return filepath.Join(".", "CLICKER", strings.ToUpper(modeID)+".txt")
}
