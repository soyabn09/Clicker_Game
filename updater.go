package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	g "github.com/AllenDang/giu"
)

const (
	releaseAPIURL = "https://api.github.com/repos/soyabn09/Game/releases/latest"
	userAgent     = "ClickerGame/" + "updater"
)

type updateState int

const (
	updateChecking updateState = iota
	updateReady
	updateRequired
	updateDownloading
	updateRestarting
	updateFailed
)

type githubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type updaterStatus struct {
	State   updateState
	Message string
	Latest  string
}

var (
	updaterMu     sync.Mutex
	currentUpdate = updaterStatus{State: updateChecking, Message: "Checking GitHub for updates..."}
)

func canUseGame() bool {
	updaterMu.Lock()
	defer updaterMu.Unlock()
	return currentUpdate.State == updateReady
}

func updaterSnapshot() updaterStatus {
	updaterMu.Lock()
	defer updaterMu.Unlock()
	return currentUpdate
}

func setUpdaterStatus(state updateState, latest, message string) {
	updaterMu.Lock()
	currentUpdate = updaterStatus{State: state, Latest: latest, Message: message}
	updaterMu.Unlock()
	g.Update()
}

func checkForUpdates() {
	setUpdaterStatus(updateChecking, "", "Checking GitHub for updates...")

	release, err := fetchLatestRelease()
	if err != nil {
		setUpdaterStatus(updateFailed, "", "Could not check GitHub releases: "+err.Error())
		return
	}

	if sameVersion(appVersion, release.TagName) {
		setUpdaterStatus(updateReady, release.TagName, "You are using the latest version.")
		return
	}

	setUpdaterStatus(updateRequired, release.TagName, "Version "+release.TagName+" is available. Updating is required before playing.")
	go downloadAndInstallUpdate(release)
}

func downloadAndInstallUpdate(release *githubRelease) {
	setUpdaterStatus(updateDownloading, release.TagName, "Downloading "+release.TagName+"...")

	assetURL, assetName, err := findUpdateAsset(release)
	if err != nil {
		setUpdaterStatus(updateFailed, release.TagName, err.Error())
		return
	}

	tmpPath, err := downloadAsset(assetURL, assetName)
	if err != nil {
		setUpdaterStatus(updateFailed, release.TagName, "Download failed: "+err.Error())
		return
	}

	updateBinary := tmpPath
	lowerTmp := strings.ToLower(tmpPath)
	if strings.HasSuffix(lowerTmp, ".zip") {
		updateBinary, err = extractExecutableFromZip(tmpPath)
		if err != nil {
			setUpdaterStatus(updateFailed, release.TagName, "Could not unpack update: "+err.Error())
			return
		}
	} else if strings.HasSuffix(lowerTmp, ".tar.gz") || strings.HasSuffix(lowerTmp, ".tgz") {
		updateBinary, err = extractExecutableFromTarGz(tmpPath)
		if err != nil {
			setUpdaterStatus(updateFailed, release.TagName, "Could not unpack update: "+err.Error())
			return
		}
	}

	setUpdaterStatus(updateRestarting, release.TagName, "Installing update and restarting...")
	if err := replaceAndRestart(updateBinary); err != nil {
		setUpdaterStatus(updateFailed, release.TagName, "Install failed: "+err.Error())
	}
}

func fetchLatestRelease() (*githubRelease, error) {
	req, err := http.NewRequest(http.MethodGet, releaseAPIURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)

	client := http.Client{Timeout: 20 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub returned %s", res.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(res.Body).Decode(&release); err != nil {
		return nil, err
	}
	if strings.TrimSpace(release.TagName) == "" {
		return nil, errors.New("latest release has no tag")
	}
	return &release, nil
}

func findUpdateAsset(release *githubRelease) (string, string, error) {
	candidates := assetCandidates(release.TagName)
	for _, candidate := range candidates {
		for _, asset := range release.Assets {
			if strings.EqualFold(asset.Name, candidate) && asset.BrowserDownloadURL != "" {
				return asset.BrowserDownloadURL, asset.Name, nil
			}
		}
	}

	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if asset.BrowserDownloadURL == "" {
			continue
		}
		if runtime.GOOS == "windows" && strings.HasSuffix(name, ".exe") && strings.Contains(name, "clicker") {
			return asset.BrowserDownloadURL, asset.Name, nil
		}
		if runtime.GOOS == "darwin" && (strings.HasSuffix(name, ".zip") || strings.Contains(name, "mac")) {
			return asset.BrowserDownloadURL, asset.Name, nil
		}
		if runtime.GOOS == "linux" && strings.Contains(name, "linux") {
			return asset.BrowserDownloadURL, asset.Name, nil
		}
	}

	return "", "", errors.New("no compatible update asset was found for " + runtime.GOOS)
}

func assetCandidates(version string) []string {
	version = strings.TrimSpace(version)
	if version == "" {
		version = "latest"
	}

	switch runtime.GOOS {
	case "windows":
		return []string{
			"ClickerGame-" + version + ".exe",
			"ClickerGame.exe",
			"clicker-game.exe",
			"Game.exe",
		}
	case "darwin":
		return []string{
			"ClickerGame-" + version + "-macos.zip",
			"ClickerGame-macos.zip",
			"ClickerGame.MacOS.zip",
			"ClickerGame-darwin.zip",
		}
	case "linux":
		return []string{
			"ClickerGame-" + version + "-linux.tar.gz",
			"ClickerGame-" + version + "-linux",
			"ClickerGame-linux.tar.gz",
			"ClickerGame-linux",
		}
	default:
		return nil
	}
}

func downloadAsset(url, name string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	client := http.Client{Timeout: 5 * time.Minute}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		return "", fmt.Errorf("download returned %s", res.Status)
	}

	tmpDir, err := os.MkdirTemp("", "clicker-game-update-*")
	if err != nil {
		return "", err
	}
	path := filepath.Join(tmpDir, filepath.Base(name))
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, res.Body); err != nil {
		return "", err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0o755); err != nil {
			return "", err
		}
	}
	return path, nil
}

func extractExecutableFromZip(zipPath string) (string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	tmpDir, err := os.MkdirTemp("", "clicker-game-unpack-*")
	if err != nil {
		return "", err
	}

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := filepath.Base(file.Name)
		lower := strings.ToLower(name)
		if runtime.GOOS == "windows" && !strings.HasSuffix(lower, ".exe") {
			continue
		}
		if runtime.GOOS != "windows" && strings.Contains(lower, ".") && !strings.Contains(lower, "clicker") {
			continue
		}

		source, err := file.Open()
		if err != nil {
			return "", err
		}
		defer source.Close()

		targetPath := filepath.Join(tmpDir, name)
		target, err := os.Create(targetPath)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(target, source); err != nil {
			_ = target.Close()
			return "", err
		}
		if err := target.Close(); err != nil {
			return "", err
		}
		if err := os.Chmod(targetPath, 0o755); err != nil {
			return "", err
		}
		return targetPath, nil
	}

	return "", errors.New("zip did not contain an executable")
}

func extractExecutableFromTarGz(tarPath string) (string, error) {
	file, err := os.Open(tarPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzipReader.Close()

	tmpDir, err := os.MkdirTemp("", "clicker-game-unpack-*")
	if err != nil {
		return "", err
	}

	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if header.FileInfo().IsDir() {
			continue
		}

		name := filepath.Base(header.Name)
		lower := strings.ToLower(name)
		if !strings.Contains(lower, "clicker") {
			continue
		}

		targetPath := filepath.Join(tmpDir, name)
		target, err := os.Create(targetPath)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(target, reader); err != nil {
			_ = target.Close()
			return "", err
		}
		if err := target.Close(); err != nil {
			return "", err
		}
		if err := os.Chmod(targetPath, 0o755); err != nil {
			return "", err
		}
		return targetPath, nil
	}

	return "", errors.New("archive did not contain an executable")
}

func replaceAndRestart(updatePath string) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		return replaceAndRestartWindows(updatePath, exePath)
	}

	if err := os.Rename(updatePath, exePath); err != nil {
		return err
	}
	if err := exec.Command(exePath).Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

func replaceAndRestartWindows(updatePath, exePath string) error {
	scriptPath := filepath.Join(os.TempDir(), "clicker-game-update.ps1")
	script := fmt.Sprintf(
		"Wait-Process -Id %d\nCopy-Item -LiteralPath '%s' -Destination '%s' -Force\nStart-Process -FilePath '%s'\n",
		os.Getpid(),
		escapePowerShellPath(updatePath),
		escapePowerShellPath(exePath),
		escapePowerShellPath(exePath),
	)
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		return err
	}

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

func escapePowerShellPath(path string) string {
	return strings.ReplaceAll(path, "'", "''")
}

func sameVersion(current, latest string) bool {
	return normalizeVersion(current) == normalizeVersion(latest)
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(strings.ToLower(version))
	version = strings.TrimPrefix(version, "refs/tags/")
	version = strings.TrimPrefix(version, "version-")
	version = strings.TrimPrefix(version, "v")
	return version
}
