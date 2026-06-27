package main

import (
	"fmt"

	"github.com/liup215/gline/internal/config"
	"github.com/liup215/gline/internal/log"
)

var (
	// Global config manager
	configManager *config.Manager
)

// InitConfig initializes the configuration and logging for GUI mode
func InitConfig() error {
	// Initialize configuration
	configManager = config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logging
	cfg := configManager.Get()
	logConfig := log.Config{
		Level:   cfg.Log.Level,
		File:    cfg.Log.File,
		Console: true,
		Color:   true,
	}

	if err := log.Init(logConfig); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	log.Debug("Configuration loaded successfully")
	log.Debugf("Log level: %s", log.GetLevel())

	return nil
}

// GetConfigManager returns the global config manager
func GetConfigManager() *config.Manager {
	return configManager
}
