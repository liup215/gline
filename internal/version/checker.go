// Package version provides version information and update checking for gline.
package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/liup215/gline/internal/log"
)

// Checker handles version checking against GitHub releases
type Checker struct {
	config      CheckerConfig
	current     string
	client      *http.Client
	mu          sync.RWMutex
	state       CheckerState
	lastCheck   *CachedResult
	onUpdate    func(UpdateInfo)
	onError     func(error)
}

// NewChecker creates a new version checker with the given configuration
func NewChecker(config CheckerConfig) *Checker {
	return &Checker{
		config:  config,
		current: Version,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		state: StateIdle,
	}
}

// NewDefaultChecker creates a new version checker with default configuration
func NewDefaultChecker() *Checker {
	return NewChecker(DefaultCheckerConfig())
}

// SetUpdateCallback sets a callback function to be called when an update is available
func (c *Checker) SetUpdateCallback(callback func(UpdateInfo)) {
	c.mu.Lock()
	c.onUpdate = callback
	c.mu.Unlock()
}

// SetErrorCallback sets a callback function to be called when an error occurs
func (c *Checker) SetErrorCallback(callback func(error)) {
	c.mu.Lock()
	c.onError = callback
	c.mu.Unlock()
}

// GetState returns the current state of the checker
func (c *Checker) GetState() CheckerState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// GetLastResult returns the last check result from cache
func (c *Checker) GetLastResult() *UpdateCheckResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastCheck == nil {
		return nil
	}

	// Return a copy
	result := c.lastCheck.Result
	return &result
}

// ShouldCheck returns true if enough time has passed since the last check
func (c *Checker) ShouldCheck() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastCheck == nil {
		return true
	}

	return c.lastCheck.IsExpired(c.config.CheckInterval)
}

// CheckNow performs a synchronous version check
func (c *Checker) CheckNow() (*UpdateCheckResult, error) {
	if !c.config.Enabled {
		return nil, fmt.Errorf("version checking is disabled")
	}

	c.mu.Lock()
	c.state = StateChecking
	c.mu.Unlock()

	log.Debug("Checking for updates...")

	result := c.fetchLatestRelease()

	c.mu.Lock()
	c.state = StateIdle
	if result.Error != nil {
		c.state = StateError
	}
	c.lastCheck = &CachedResult{
		Result:    *result,
		CheckedAt: time.Now(),
	}
	onUpdate := c.onUpdate
	onError := c.onError
	c.mu.Unlock()

	// Save to cache file if configured
	if c.config.CacheFile != "" && result.Error == nil {
		if err := c.saveCache(); err != nil {
			log.Warnf("Failed to save update check cache: %v", err)
		}
	}

	// Call callbacks
	if result.Error != nil && onError != nil {
		onError(result.Error)
	}

	if result.HasUpdate && onUpdate != nil {
		info := c.resultToUpdateInfo(result, UITypeCLI)
		onUpdate(info)
	}

	return result, result.Error
}

// CheckAsync performs an asynchronous version check
func (c *Checker) CheckAsync(callback func(*UpdateCheckResult)) {
	go func() {
		result, _ := c.CheckNow()
		if callback != nil {
			callback(result)
		}
	}()
}

// CheckAsyncWithUI performs an asynchronous version check with UI-specific formatting
func (c *Checker) CheckAsyncWithUI(uiType UIType, callback func(UpdateInfo)) {
	go func() {
		result, _ := c.CheckNow()
		if callback != nil {
			info := c.resultToUpdateInfo(result, uiType)
			callback(info)
		}
	}()
}

// GetUpdateInfoForCLI returns update information formatted for CLI display
func (c *Checker) GetUpdateInfoForCLI() (*UpdateInfo, error) {
	result, err := c.CheckNow()
	if err != nil {
		return nil, err
	}

	info := c.resultToUpdateInfo(result, UITypeCLI)
	return &info, nil
}

// GetUpdateInfoForGUI returns update information formatted for GUI display
func (c *Checker) GetUpdateInfoForGUI() (*UpdateInfo, error) {
	result, err := c.CheckNow()
	if err != nil {
		return nil, err
	}

	info := c.resultToUpdateInfo(result, UITypeGUI)
	return &info, nil
}

// LoadCache loads the cached check result from file
func (c *Checker) LoadCache() error {
	if c.config.CacheFile == "" {
		return nil
	}

	data, err := os.ReadFile(c.config.CacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Cache doesn't exist yet
		}
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	var cached CachedResult
	if err := json.Unmarshal(data, &cached); err != nil {
		return fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	c.mu.Lock()
	c.lastCheck = &cached
	c.mu.Unlock()

	log.Debug("Loaded update check cache")
	return nil
}

// fetchLatestRelease fetches the latest release from GitHub
func (c *Checker) fetchLatestRelease() *UpdateCheckResult {
	result := &UpdateCheckResult{
		CurrentVersion: c.current,
		CheckedAt:      time.Now(),
	}

	// Create request
	req, err := http.NewRequest("GET", c.config.GitHubAPIURL, nil)
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		return result
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "gline-version-checker")

	// Perform request
	resp, err := c.client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch release: %w", err)
		return result
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
		return result
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("failed to read response body: %w", err)
		return result
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		result.Error = fmt.Errorf("failed to parse release JSON: %w", err)
		return result
	}

	// Parse and compare versions
	result.LatestVersion = normalizeVersion(release.TagName)
	result.ReleaseURL = release.HTMLURL
	result.ReleaseNotes = release.Body
	result.PublishedAt = release.PublishedAt

	// Find appropriate download URL
	result.DownloadURL = c.findDownloadURL(&release)

	// Compare versions
	currentParsed := ParseVersion(c.current)
	latestParsed := ParseVersion(result.LatestVersion)

	result.HasUpdate = latestParsed.GreaterThan(currentParsed)

	if result.HasUpdate {
		log.Infof("New version available: %s (current: %s)", result.LatestVersion, result.CurrentVersion)
	} else {
		log.Debugf("No update available (current: %s, latest: %s)", result.CurrentVersion, result.LatestVersion)
	}

	return result
}

// findDownloadURL finds the appropriate download URL for the current platform
func (c *Checker) findDownloadURL(release *GitHubRelease) string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map Go arch to common naming conventions
	archMap := map[string]string{
		"amd64": "amd64",
		"arm64": "arm64",
		"386":   "386",
	}

	arch := archMap[goarch]
	if arch == "" {
		arch = goarch
	}

	// Look for matching asset
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)

		// Check OS
		if !strings.Contains(name, goos) {
			continue
		}

		// Check architecture
		if !strings.Contains(name, arch) {
			continue
		}

		return asset.BrowserDownloadURL
	}

	// If no specific match, return the release page
	return release.HTMLURL
}

// saveCache saves the cached result to file
func (c *Checker) saveCache() error {
	if c.config.CacheFile == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(c.config.CacheFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	c.mu.RLock()
	cached := c.lastCheck
	c.mu.RUnlock()

	if cached == nil {
		return nil
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(c.config.CacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// resultToUpdateInfo converts a check result to UI-formatted update info
func (c *Checker) resultToUpdateInfo(result *UpdateCheckResult, uiType UIType) UpdateInfo {
	info := UpdateInfo{
		HasUpdate:      result.HasUpdate,
		CurrentVersion: result.CurrentVersion,
		LatestVersion:  result.LatestVersion,
		ReleaseURL:     result.ReleaseURL,
		DownloadURL:    result.DownloadURL,
		ReleaseNotes:   result.ReleaseNotes,
		PublishedAt:    result.PublishedAt,
		UIType:         uiType,
	}

	// Format message based on UI type
	if result.HasUpdate {
		switch uiType {
		case UITypeCLI:
			info.Message = fmt.Sprintf(
				"A new version of gline is available!\n\n"+
					"Current version: %s\n"+
					"Latest version:  %s\n\n"+
					"Release notes:\n%s\n\n"+
					"Download: %s",
				result.CurrentVersion,
				result.LatestVersion,
				truncateString(result.ReleaseNotes, 500),
				result.DownloadURL,
			)
		case UITypeGUI:
			info.Message = fmt.Sprintf(
				"Version %s is now available (you have %s)",
				result.LatestVersion,
				result.CurrentVersion,
			)
		}
	} else {
		info.Message = fmt.Sprintf("You are running the latest version (%s)", result.CurrentVersion)
	}

	return info
}

// ParseVersion parses a semantic version string into a VersionInfo struct
func ParseVersion(v string) VersionInfo {
	v = normalizeVersion(v)

	// Remove 'v' prefix if present
	v = strings.TrimPrefix(v, "v")

	// Split by pre-release separator
	parts := strings.SplitN(v, "-", 2)
	versionPart := parts[0]
	pre := ""
	if len(parts) > 1 {
		pre = parts[1]
	}

	// Parse version numbers
	var major, minor, patch int
	versionRegex := regexp.MustCompile(`^(\d+)(?:\.(\d+))?(?:\.(\d+))?`)
	matches := versionRegex.FindStringSubmatch(versionPart)

	if len(matches) > 1 {
		major, _ = strconv.Atoi(matches[1])
	}
	if len(matches) > 2 && matches[2] != "" {
		minor, _ = strconv.Atoi(matches[2])
	}
	if len(matches) > 3 && matches[3] != "" {
		patch, _ = strconv.Atoi(matches[3])
	}

	return VersionInfo{
		Major: major,
		Minor: minor,
		Patch: patch,
		Pre:   pre,
	}
}

// normalizeVersion normalizes a version string
func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	return v
}

// GreaterThan returns true if this version is greater than another
func (v VersionInfo) GreaterThan(other VersionInfo) bool {
	// Compare major version
	if v.Major != other.Major {
		return v.Major > other.Major
	}

	// Compare minor version
	if v.Minor != other.Minor {
		return v.Minor > other.Minor
	}

	// Compare patch version
	if v.Patch != other.Patch {
		return v.Patch > other.Patch
	}

	// Handle pre-release versions
	// A version without pre-release is greater than one with
	if v.Pre == "" && other.Pre != "" {
		return true
	}
	if v.Pre != "" && other.Pre == "" {
		return false
	}

	// Compare pre-release strings
	if v.Pre != other.Pre {
		return v.Pre > other.Pre
	}

	return false
}

// String returns the string representation of the version
func (v VersionInfo) String() string {
	result := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		result += "-" + v.Pre
	}
	return result
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// IsDevVersion returns true if the current version is a development version
func IsDevVersion() bool {
	return Version == "dev" || Version == "unknown"
}

// CheckForUpdates is a convenience function to check for updates with default settings
func CheckForUpdates() (*UpdateCheckResult, error) {
	checker := NewDefaultChecker()
	if err := checker.LoadCache(); err != nil {
		log.Warnf("Failed to load update cache: %v", err)
	}

	// Only check if enough time has passed
	if !checker.ShouldCheck() {
		log.Debug("Skipping update check - too soon since last check")
		return checker.GetLastResult(), nil
	}

	return checker.CheckNow()
}

// CheckForUpdatesAsync is a convenience function to check for updates asynchronously
func CheckForUpdatesAsync(callback func(*UpdateCheckResult)) {
	checker := NewDefaultChecker()
	if err := checker.LoadCache(); err != nil {
		log.Warnf("Failed to load update cache: %v", err)
	}

	if !checker.ShouldCheck() {
		log.Debug("Skipping update check - too soon since last check")
		if callback != nil {
			callback(checker.GetLastResult())
		}
		return
	}

	checker.CheckAsync(callback)
}
