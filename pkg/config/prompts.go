package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolvePrompt resolves the final prompt content based on priority:
// 1. CLI parameter file path (highest priority)
// 2. Config file setting (filename in prompts/ directory)
// 3. Hardcoded default template (lowest priority)
//
// Parameters:
//   - configDir: The directory where config.yaml is located
//   - configFile: The filename configured in config.yaml (e.g., "commit-message.txt")
//   - cliFile: The file path specified via CLI parameter (can be absolute or relative)
//   - defaultTemplate: The hardcoded default template content
//
// Returns: The final prompt content to use
func ResolvePrompt(configDir, configFile, cliFile, defaultTemplate string) (string, error) {
	// 1. First priority: CLI parameter specified file path
	if cliFile != "" {
		content, err := os.ReadFile(cliFile)
		if err != nil {
			return "", fmt.Errorf("failed to read CLI-specified prompt file %s: %w", cliFile, err)
		}
		return string(content), nil
	}

	// 2. Second priority: Config file setting (filename in prompts/ directory)
	if configFile != "" {
		promptPath := filepath.Join(configDir, "prompts", configFile)
		content, err := os.ReadFile(promptPath)
		if err != nil {
			return "", fmt.Errorf("failed to read configured prompt file %s: %w", promptPath, err)
		}
		return string(content), nil
	}

	// 3. Third priority: Hardcoded default template
	return defaultTemplate, nil
}

// GetConfigDir returns the directory where config.yaml is located.
// This is used by the main package to get the config directory for prompt resolution.
func GetConfigDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to determine executable path: %w", err)
	}
	binaryName := filepath.Base(exePath)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", binaryName), nil
}
