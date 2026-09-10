package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	repoOwner     = "Dolyyyy"
	repoName      = "deck"
	cacheDuration = 4 * time.Hour
)

// ReleaseInfo represents GitHub release metadata.
type ReleaseInfo struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Body        string    `json:"body"`
	Assets      []Asset   `json:"assets"`
}

// Asset represents a release asset binary archive.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type cachedCheck struct {
	LastChecked time.Time    `json:"last_checked"`
	Latest      *ReleaseInfo `json:"latest"`
}

// Updater handles checking for new releases and performing self-updates.
type Updater struct {
	cachePath string
	client    *http.Client
}

// NewUpdater creates a new updater with local cache support.
func NewUpdater(configDir string) *Updater {
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config", "deck")
	}
	return &Updater{
		cachePath: filepath.Join(configDir, ".update_cache.json"),
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// CheckUpdate checks GitHub for a newer version than currentVersion.
func (u *Updater) CheckUpdate(currentVersion string, force bool) (*ReleaseInfo, bool, error) {
	if !force {
		if cached, ok := u.readCache(); ok {
			if time.Since(cached.LastChecked) < cacheDuration && cached.Latest != nil {
				hasNew := isNewer(cached.Latest.TagName, currentVersion)
				return cached.Latest, hasNew, nil
			}
		}
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "deck-updater/"+currentVersion)

	resp, err := u.client.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var release ReleaseInfo
		if err := json.NewDecoder(resp.Body).Decode(&release); err == nil {
			u.writeCache(&release)
			hasNew := isNewer(release.TagName, currentVersion)
			return &release, hasNew, nil
		}
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Fallback to web redirect (100% immune to GitHub API 403 Forbidden rate limits)
	rel, redirectErr := u.checkRedirectVersion()
	if redirectErr == nil && rel != nil {
		u.writeCache(rel)
		hasNew := isNewer(rel.TagName, currentVersion)
		return rel, hasNew, nil
	}

	if err != nil {
		return nil, false, err
	}
	return nil, false, fmt.Errorf("failed to check for updates: %w", redirectErr)
}

func (u *Updater) checkRedirectVersion() (*ReleaseInfo, error) {
	redirectClient := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := redirectClient.Get(fmt.Sprintf("https://github.com/%s/%s/releases/latest", repoOwner, repoName))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	loc := resp.Header.Get("Location")
	if loc == "" {
		return nil, fmt.Errorf("no release redirect found")
	}

	idx := strings.LastIndex(loc, "/")
	if idx == -1 || idx+1 >= len(loc) {
		return nil, fmt.Errorf("invalid release redirect url: %s", loc)
	}

	tagName := loc[idx+1:]
	vNum := strings.TrimPrefix(tagName, "v")

	assets := []Asset{
		{
			Name:               fmt.Sprintf("deck_%s_linux_amd64.tar.gz", vNum),
			BrowserDownloadURL: fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/deck_%s_linux_amd64.tar.gz", repoOwner, repoName, tagName, vNum),
		},
		{
			Name:               fmt.Sprintf("deck_%s_linux_arm64.tar.gz", vNum),
			BrowserDownloadURL: fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/deck_%s_linux_arm64.tar.gz", repoOwner, repoName, tagName, vNum),
		},
		{
			Name:               fmt.Sprintf("deck_%s_darwin_amd64.tar.gz", vNum),
			BrowserDownloadURL: fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/deck_%s_darwin_amd64.tar.gz", repoOwner, repoName, tagName, vNum),
		},
		{
			Name:               fmt.Sprintf("deck_%s_darwin_arm64.tar.gz", vNum),
			BrowserDownloadURL: fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/deck_%s_darwin_arm64.tar.gz", repoOwner, repoName, tagName, vNum),
		},
		{
			Name:               fmt.Sprintf("deck_%s_windows_amd64.zip", vNum),
			BrowserDownloadURL: fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/deck_%s_windows_amd64.zip", repoOwner, repoName, tagName, vNum),
		},
	}

	return &ReleaseInfo{
		TagName: tagName,
		Name:    tagName,
		Assets:  assets,
	}, nil
}

// SelfUpdate downloads and replaces the running binary with the latest release.
func (u *Updater) SelfUpdate(currentVersion string) (string, error) {
	release, hasNew, err := u.CheckUpdate(currentVersion, true)
	if err != nil {
		return "", fmt.Errorf("failed to check for updates: %w", err)
	}
	if !hasNew {
		return release.TagName, fmt.Errorf("deck is already at latest version (%s)", currentVersion)
	}

	// Find asset for current OS/Arch
	// Example goreleaser naming: deck_0.2.0_Linux_x86_64.tar.gz or deck_0.2.0_Darwin_arm64.tar.gz
	targetOS := runtime.GOOS
	targetArch := runtime.GOARCH

	var matchingAsset *Asset
	for _, a := range release.Assets {
		name := strings.ToLower(a.Name)
		if strings.Contains(name, strings.ToLower(targetOS)) {
			if targetArch == "amd64" && (strings.Contains(name, "x86_64") || strings.Contains(name, "amd64")) {
				matchingAsset = &a
				break
			} else if targetArch == "arm64" && (strings.Contains(name, "arm64") || strings.Contains(name, "aarch64")) {
				matchingAsset = &a
				break
			}
		}
	}

	if matchingAsset == nil {
		return release.TagName, fmt.Errorf("no compatible release binary found for %s/%s", targetOS, targetArch)
	}

	// Download archive
	resp, err := u.client.Get(matchingAsset.BrowserDownloadURL)
	if err != nil {
		return release.TagName, fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return release.TagName, fmt.Errorf("failed to read binary payload: %w", err)
	}

	// Extract binary
	var binaryBytes []byte
	if strings.HasSuffix(matchingAsset.Name, ".tar.gz") || strings.HasSuffix(matchingAsset.Name, ".tgz") {
		binaryBytes, err = extractTarGz(data, "deck")
	} else if strings.HasSuffix(matchingAsset.Name, ".zip") {
		binaryBytes, err = extractZip(data, "deck")
	} else {
		binaryBytes = data
	}
	if err != nil {
		return release.TagName, fmt.Errorf("failed to extract binary: %w", err)
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return release.TagName, fmt.Errorf("failed to locate current executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return release.TagName, fmt.Errorf("failed to resolve symlink: %w", err)
	}

	// Write to temporary replacement file in the same directory
	tempFile := execPath + ".new"
	if err := os.WriteFile(tempFile, binaryBytes, 0755); err != nil {
		return release.TagName, fmt.Errorf("failed to write update binary (permission denied? try sudo): %w", err)
	}

	// Atomic replace
	if err := os.Rename(tempFile, execPath); err != nil {
		_ = os.Remove(tempFile)
		return release.TagName, fmt.Errorf("failed to replace executable: %w", err)
	}

	return release.TagName, nil
}

func extractTarGz(data []byte, targetName string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		base := filepath.Base(hdr.Name)
		if base == targetName || base == targetName+".exe" {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("binary %q not found in tar archive", targetName)
}

func extractZip(data []byte, targetName string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	for _, f := range zr.File {
		base := filepath.Base(f.Name)
		if base == targetName || base == targetName+".exe" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("binary %q not found in zip archive", targetName)
}

func (u *Updater) readCache() (*cachedCheck, bool) {
	data, err := os.ReadFile(u.cachePath)
	if err != nil {
		return nil, false
	}
	var c cachedCheck
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, false
	}
	return &c, true
}

func (u *Updater) writeCache(rel *ReleaseInfo) {
	_ = os.MkdirAll(filepath.Dir(u.cachePath), 0700)
	c := cachedCheck{
		LastChecked: time.Now(),
		Latest:      rel,
	}
	data, _ := json.Marshal(c)
	_ = os.WriteFile(u.cachePath, data, 0600)
}

// isNewer compares two semver strings (e.g., "v0.2.0" vs "v0.1.0").
func isNewer(latest, current string) bool {
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")

	lParts := strings.Split(latest, ".")
	cParts := strings.Split(current, ".")

	for i := 0; i < len(lParts) && i < len(cParts); i++ {
		lNum, err1 := strconv.Atoi(lParts[i])
		cNum, err2 := strconv.Atoi(cParts[i])
		if err1 == nil && err2 == nil {
			if lNum > cNum {
				return true
			}
			if lNum < cNum {
				return false
			}
		}
	}

	return len(lParts) > len(cParts)
}
