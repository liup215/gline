// Package version provides version information and update checking for gline.
package version

import (
	"time"
)

// GitHubRelease represents a release from the GitHub API
type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

// Asset represents a downloadable asset in a GitHub release
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int    `json:"size"`
	ContentType        string `json:"content_type"`
}

// UpdateCheckResult represents the result of an update check
type UpdateCheckResult struct {
	// HasUpdate indicates if a newer version is available
	HasUpdate bool

	// CurrentVersion is the currently running version
	CurrentVersion string

	// LatestVersion is the latest version available
	LatestVersion string

	// ReleaseURL is the URL to the release page
	ReleaseURL string

	// DownloadURL is the direct download URL for the appropriate asset
	DownloadURL string

	// ReleaseNotes contains the release notes/body
	ReleaseNotes string

	// PublishedAt is when the release was published
	PublishedAt time.Time

	// CheckedAt is when the check was performed
	CheckedAt time.Time

	// Error contains any error that occurred during the check
	Error error
}

// CheckerConfig holds configuration for the version checker
type CheckerConfig struct {
	// Enabled controls whether update checking is enabled
	Enabled bool

	// CheckInterval is the minimum time between checks
	CheckInterval time.Duration

	// GitHubAPIURL is the URL for the GitHub API
	GitHubAPIURL string

	// Timeout for HTTP requests
	Timeout time.Duration

	// CacheFile is the path to cache the last check result
	CacheFile string
}

// DefaultCheckerConfig returns the default configuration
func DefaultCheckerConfig() CheckerConfig {
	return CheckerConfig{
		Enabled:       true,
		CheckInterval: 24 * time.Hour,
		GitHubAPIURL:  "https://api.github.com/repos/liup215/gline/releases/latest",
		Timeout:       30 * time.Second,
		CacheFile:     "",
	}
}

// CachedResult represents a cached update check result
type CachedResult struct {
	Result    UpdateCheckResult
	CheckedAt time.Time
}

// IsExpired returns true if the cached result is older than the check interval
func (c *CachedResult) IsExpired(interval time.Duration) bool {
	return time.Since(c.CheckedAt) > interval
}

// VersionInfo holds parsed version information
type VersionInfo struct {
	Major int
	Minor int
	Patch int
	Pre   string // pre-release suffix
}

// UIType represents the type of UI being used
type UIType string

const (
	// UITypeCLI represents command-line interface mode
	UITypeCLI UIType = "cli"

	// UITypeGUI represents graphical user interface mode
	UITypeGUI UIType = "gui"
)

// UpdateInfo provides formatted information for UI display
type UpdateInfo struct {
	// HasUpdate indicates if a newer version is available
	HasUpdate bool

	// CurrentVersion is the currently running version
	CurrentVersion string

	// LatestVersion is the latest version available
	LatestVersion string

	// Message is a user-friendly message about the update
	Message string

	// ReleaseURL is the URL to view release details
	ReleaseURL string

	// DownloadURL is the URL to download the update
	DownloadURL string

	// ReleaseNotes contains the release notes
	ReleaseNotes string

	// PublishedAt is when the release was published
	PublishedAt time.Time

	// UIType indicates which UI mode this info is formatted for
	UIType UIType
}

// CheckerState represents the current state of the version checker
type CheckerState int

const (
	// StateIdle means the checker is not currently checking
	StateIdle CheckerState = 0

	// StateChecking means a check is in progress
	StateChecking CheckerState = 1

	// StateError means the last check resulted in an error
	StateError CheckerState = 2
)

// String returns a string representation of the state
func (s CheckerState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateChecking:
		return "checking"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}
