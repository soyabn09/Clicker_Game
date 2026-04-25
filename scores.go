package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type scoreStore struct {
	Version   int            `json:"version"`
	HighScore map[string]int `json:"high_scores"`
}

var (
	scoresMu sync.Mutex
	scores   = scoreStore{
		Version:   1,
		HighScore: defaultHighScores(),
	}

	lastScoreFileCheck time.Time
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
		scores.HighScore = defaultHighScores()
		return saveScoresLocked()
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, &scores); err != nil {
		scores.HighScore = defaultHighScores()
		return saveScoresLocked()
	}
	if scores.HighScore == nil {
		scores.HighScore = map[string]int{}
	}

	changed := false
	for _, id := range []string{"easy", "medium", "hard", "custom:60"} {
		if _, ok := scores.HighScore[id]; !ok {
			scores.HighScore[id] = 0
			changed = true
		}
	}
	if _, ok := scores.HighScore["custom"]; ok {
		delete(scores.HighScore, "custom")
		changed = true
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

func defaultHighScores() map[string]int {
	return map[string]int{"easy": 0, "medium": 0, "hard": 0, "custom:60": 0}
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

func ensureScoresFile() error {
	scoresMu.Lock()
	defer scoresMu.Unlock()

	now := time.Now()
	if now.Sub(lastScoreFileCheck) < time.Second {
		return nil
	}
	lastScoreFileCheck = now

	path, err := scoresPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
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
