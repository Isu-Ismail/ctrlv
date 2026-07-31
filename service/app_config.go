package service

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type AppConfig struct {
	APIKey       string `json:"api_key"`
	Model        string `json:"model"`
	CustomPrompt string `json:"custom_prompt,omitempty"`
	RelayURL     string `json:"relay_url"`
	Editor       string `json:"editor,omitempty"`
}

func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	ctrlvDir := filepath.Join(home, ".ctrlv")
	_ = os.MkdirAll(ctrlvDir, 0755)
	return filepath.Join(ctrlvDir, "ctrlv_config.json")
}

func EnsureConfigExists() (string, *AppConfig, error) {
	configPath := GetConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultCfg := &AppConfig{
			APIKey:       "",
			Model:        "openrouter/auto",
			CustomPrompt: "Solve the problem shown in this screenshot. Output ONLY clean, working code without explanations or markdown formatting.",
			RelayURL:     "wss://ctrlv.onrender.com/ws",
			Editor:       "",
		}
		if err := SaveAppConfig(defaultCfg); err != nil {
			return configPath, defaultCfg, fmt.Errorf("failed to create default config at %s: %w", configPath, err)
		}
		return configPath, defaultCfg, nil
	}

	cfg, err := LoadAppConfig()
	if err != nil {
		return configPath, nil, err
	}
	return configPath, cfg, nil
}

func LoadAppConfig() (*AppConfig, error) {
	configPath := GetConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w", configPath, err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	if cfg.Model == "" {
		cfg.Model = "openrouter/auto"
	}
	if strings.TrimSpace(cfg.CustomPrompt) == "" {
		cfg.CustomPrompt = "Solve the problem shown in this screenshot. Output ONLY clean, working code without explanations or markdown formatting."
	}

	return &cfg, nil
}

func SaveAppConfig(cfg *AppConfig) error {
	configPath := GetConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func (c *AppConfig) ToAIConfig() *AIConfig {
	prompt := strings.TrimSpace(c.CustomPrompt)
	if prompt == "" {
		prompt = "Solve the problem shown in this screenshot. Output ONLY clean, working code without explanations or markdown formatting."
	}
	return &AIConfig{
		Provider:     "openrouter",
		APIKey:       c.APIKey,
		Model:        c.Model,
		CustomPrompt: prompt,
		CodeOnly:     true,
	}
}

func OpenConfigInEditor(editorOverride string) error {
	configPath, cfg, err := EnsureConfigExists()
	if err != nil {
		return fmt.Errorf("unable to ensure config file exists: %w", err)
	}

	if editorOverride != "" {
		cfg.Editor = strings.TrimSpace(editorOverride)
		if err := SaveAppConfig(cfg); err != nil {
			fmt.Printf("[Warning] Failed to save editor preference to config: %v\n", err)
		} else {
			fmt.Printf("[Config] Saved preferred editor: '%s'\n", cfg.Editor)
		}
	}

	editorToUse := strings.TrimSpace(cfg.Editor)

	if editorToUse != "" {
		err := runEditorCommand(editorToUse, configPath)
		if err == nil {
			return nil
		}
		fmt.Printf("[Notice] Configured editor '%s' was not found or failed to start: %v\nFalling back to default system editor...\n", editorToUse, err)
	}

	return runDefaultSystemEditor(configPath)
}

func runEditorCommand(editorName string, filePath string) error {
	execPath, err := exec.LookPath(editorName)
	if err != nil {
		return err
	}

	cmd := exec.Command(execPath, filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if runtime.GOOS == "windows" {
		return cmd.Start()
	}
	return cmd.Run()
}

func runDefaultSystemEditor(filePath string) error {
	var fallbackCmd string
	if runtime.GOOS == "windows" {
		fallbackCmd = "notepad"
	} else {
		fallbackCmd = os.Getenv("EDITOR")
		if fallbackCmd == "" {
			fallbackCmd = "nano"
		}
	}

	execPath, err := exec.LookPath(fallbackCmd)
	if err != nil {
		if runtime.GOOS != "windows" {
			for _, alt := range []string{"vim", "vi", "xdg-open"} {
				if altPath, errAlt := exec.LookPath(alt); errAlt == nil {
					execPath = altPath
					fallbackCmd = alt
					break
				}
			}
		}
		if execPath == "" {
			return fmt.Errorf("no default text editor found to open %s. Please edit file manually at: %s", filePath, filePath)
		}
	}

	fmt.Printf("[Config] Opening %s in %s...\n", filePath, fallbackCmd)
	cmd := exec.Command(execPath, filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if runtime.GOOS == "windows" {
		return cmd.Start()
	}
	return cmd.Run()
}
