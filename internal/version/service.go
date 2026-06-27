// Package version provides version information and update checking for gline.
package version

import (
	"path/filepath"
	"time"

	"github.com/liup215/gline/internal/config"
	"github.com/liup215/gline/internal/log"
)

// Service provides version checking as a service
type Service struct {
	checker *Checker
	config  *config.UpdateConfig
}

// NewService creates a new version checking service
func NewService(cfg *config.UpdateConfig, configDir string) *Service {
	// Parse check interval
	checkInterval := 24 * time.Hour
	if cfg.CheckInterval != "" {
		if d, err := time.ParseDuration(cfg.CheckInterval); err == nil {
			checkInterval = d
		}
	}

	checkerConfig := CheckerConfig{
		Enabled:       cfg.Enabled,
		CheckInterval: checkInterval,
		GitHubAPIURL:  "https://api.github.com/repos/liup215/gline/releases/latest",
		Timeout:       30 * time.Second,
		CacheFile:     filepath.Join(configDir, "update_cache.json"),
	}

	return &Service{
		checker: NewChecker(checkerConfig),
		config:  cfg,
	}
}

// CheckForUpdates checks for updates and returns the result
func (s *Service) CheckForUpdates() (*UpdateCheckResult, error) {
	if !s.config.Enabled {
		log.Debug("Update checking is disabled")
		return nil, nil
	}

	// Load cache
	if err := s.checker.LoadCache(); err != nil {
		log.Warnf("Failed to load update cache: %v", err)
	}

	// Check if we should check
	if !s.checker.ShouldCheck() {
		log.Debug("Skipping update check - too soon since last check")
		return s.checker.GetLastResult(), nil
	}

	return s.checker.CheckNow()
}

// CheckForUpdatesAsync checks for updates asynchronously
func (s *Service) CheckForUpdatesAsync(callback func(*UpdateCheckResult)) {
	if !s.config.Enabled {
		log.Debug("Update checking is disabled")
		if callback != nil {
			callback(nil)
		}
		return
	}

	// Load cache
	if err := s.checker.LoadCache(); err != nil {
		log.Warnf("Failed to load update cache: %v", err)
	}

	// Check if we should check
	if !s.checker.ShouldCheck() {
		log.Debug("Skipping update check - too soon since last check")
		if callback != nil {
			callback(s.checker.GetLastResult())
		}
		return
	}

	s.checker.CheckAsync(callback)
}

// GetLastResult returns the last check result
func (s *Service) GetLastResult() *UpdateCheckResult {
	return s.checker.GetLastResult()
}

// ShouldCheck returns true if an update check should be performed
func (s *Service) ShouldCheck() bool {
	return s.checker.ShouldCheck()
}

// GetCurrentVersion returns the current application version
func (s *Service) GetCurrentVersion() string {
	return Version
}

// GetUpdateInfoForGUI returns update info formatted for GUI
func (s *Service) GetUpdateInfoForGUI() (*UpdateInfo, error) {
	return s.checker.GetUpdateInfoForGUI()
}

// GetUpdateInfoForCLI returns update info formatted for CLI
func (s *Service) GetUpdateInfoForCLI() (*UpdateInfo, error) {
	return s.checker.GetUpdateInfoForCLI()
}

// IsEnabled returns whether update checking is enabled
func (s *Service) IsEnabled() bool {
	return s.config.Enabled
}

// SetEnabled enables or disables update checking
func (s *Service) SetEnabled(enabled bool) {
	s.config.Enabled = enabled
}
