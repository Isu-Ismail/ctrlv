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
	Provider     string `json:"provider,omitempty"` // "google", "openrouter", "openai", "groq"
	APIKey       string `json:"api_key"`            // Your AI Provider API Key
	Model        string `json:"model"`              // Model name (e.g. "openrouter/auto", "gemini-2.0-flash", "gpt-4o-mini", "llama-3.2-11b-vision-preview")
	CustomPrompt string `json:"custom_prompt,omitempty"`
	MaxTokens    int    `json:"max_tokens,omitempty"`
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
			Provider:     "openrouter",
			APIKey:       "",
			Model:        "openrouter/auto",
			CustomPrompt: "Solve the problem shown in this screenshot. Output ONLY clean, working code without explanations or markdown formatting.",
			MaxTokens:    2048,
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

	if strings.TrimSpace(cfg.Provider) == "" {
		cfg.Provider = "openrouter"
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = "openrouter/auto"
	}
	if strings.TrimSpace(cfg.CustomPrompt) == "" {
		cfg.CustomPrompt = "Solve the problem shown in this screenshot. Output ONLY clean, working code without explanations or markdown formatting."
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 2048
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
	maxTok := c.MaxTokens
	if maxTok <= 0 {
		maxTok = 2048
	}

	provider := strings.ToLower(strings.TrimSpace(c.Provider))
	apiKey := strings.TrimSpace(c.APIKey)

	// Auto-detect provider if missing or set to "auto"
	if provider == "" || provider == "auto" {
		if strings.HasPrefix(apiKey, "gsk_") {
			provider = "groq"
		} else if strings.HasPrefix(apiKey, "AIza") {
			provider = "google"
		} else if strings.HasPrefix(apiKey, "sk-proj-") || (strings.HasPrefix(apiKey, "sk-") && !strings.HasPrefix(apiKey, "sk-or-")) {
			provider = "openai"
		} else {
			provider = "openrouter"
		}
	}

	model := strings.TrimSpace(c.Model)
	if model == "" {
		switch provider {
		case "google":
			model = "gemini-2.0-flash"
		case "openai":
			model = "gpt-4o-mini"
		case "groq":
			model = "llama-3.2-11b-vision-preview"
		default:
			model = "openrouter/auto"
		}
	}

	return &AIConfig{
		Provider:     provider,
		APIKey:       c.APIKey,
		Model:        model,
		CustomPrompt: prompt,
		MaxTokens:    maxTok,
		CodeOnly:     true,
	}
}

func OpenConfigInEditor(editorOverride string) error {
	configPath := GetConfigPath()

	// Ensure directory & file exist on disk even if corrupted
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultCfg := &AppConfig{
			Provider:     "openrouter",
			APIKey:       "",
			Model:        "openrouter/auto",
			CustomPrompt: "Solve the problem shown in this screenshot. Output ONLY clean, working code without explanations or markdown formatting.",
			MaxTokens:    2048,
			RelayURL:     "wss://ctrlv.onrender.com/ws",
			Editor:       "",
		}
		_ = SaveAppConfig(defaultCfg)
	}

	cfg, parseErr := LoadAppConfig()
	if parseErr != nil {
		fmt.Printf("[Notice] Config file has formatting issues (%v).\n[Notice] Opening %s in editor so you can fix it...\n", parseErr, configPath)
	}

	if editorOverride != "" {
		if cfg != nil {
			cfg.Editor = strings.TrimSpace(editorOverride)
			_ = SaveAppConfig(cfg)
			fmt.Printf("[Config] Saved preferred editor: '%s'\n", cfg.Editor)
		}
	}

	var editorToUse string
	if cfg != nil && strings.TrimSpace(cfg.Editor) != "" {
		editorToUse = strings.TrimSpace(cfg.Editor)
	} else if editorOverride != "" {
		editorToUse = strings.TrimSpace(editorOverride)
	}

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
