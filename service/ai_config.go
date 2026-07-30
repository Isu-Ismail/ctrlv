package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AIConfig struct {
	Provider     string `json:"provider"`      // "openrouter", "groq", "gemini"
	APIKey       string `json:"api_key"`       // API key string
	Model        string `json:"model"`         // e.g. "openrouter/auto", "google/gemini-2.0-flash-exp:free", "gemini-2.0-flash"
	CustomPrompt string `json:"custom_prompt"` // Instructions for vision AI
	CodeOnly     bool   `json:"code_only"`     // Strip markdown code fences
}

func GetAIConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	ctrlvDir := filepath.Join(home, ".ctrlv")
	_ = os.MkdirAll(ctrlvDir, 0755)
	return filepath.Join(ctrlvDir, "ai_config.json")
}

func LoadAIConfig() (*AIConfig, error) {
	configPath := GetAIConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Create default config file template if not present
		defaultCfg := &AIConfig{
			Provider:     "openrouter",
			APIKey:       "",
			Model:        "openrouter/auto",
			CustomPrompt: "Solve the coding question or problem shown in this screenshot. Output ONLY clean, working code without explanations or markdown formatting.",
			CodeOnly:     true,
		}
		SaveAIConfig(defaultCfg)
		return defaultCfg, fmt.Errorf("config file created at %s. Please edit file to paste your AI API Key", configPath)
	}

	var cfg AIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse AI config file %s: %w", configPath, err)
	}

	if cfg.Provider == "" {
		cfg.Provider = "openrouter"
	}
	if cfg.Model == "" {
		cfg.Model = "openrouter/auto"
	}
	if cfg.CustomPrompt == "" {
		cfg.CustomPrompt = "Solve the coding question or problem shown in this screenshot. Output ONLY clean, working code without explanations or markdown formatting."
	}

	return &cfg, nil
}

func SaveAIConfig(cfg *AIConfig) error {
	configPath := GetAIConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// TestAICredentials sends a lightweight test ping to verify API Key & credentials
func TestAICredentials(cfg *AIConfig) error {
	apiKey := strings.TrimSpace(cfg.APIKey)
	apiKey = strings.ReplaceAll(apiKey, "\"", "")
	apiKey = strings.ReplaceAll(apiKey, "'", "")

	if apiKey == "" {
		return fmt.Errorf("AI API Key is empty in %s", GetAIConfigPath())
	}

	client := &http.Client{Timeout: 8 * time.Second}

	switch cfg.Provider {
	case "openrouter":
		endpoint := "https://openrouter.ai/api/v1/chat/completions"
		reqBody := map[string]interface{}{
			"model": cfg.Model,
			"messages": []map[string]string{
				{"role": "user", "content": "ping"},
			},
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("OpenRouter test request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("OpenRouter auth failed (HTTP %d): %s", resp.StatusCode, string(b))
		}

	case "groq":
		endpoint := "https://api.groq.com/openai/v1/chat/completions"
		reqBody := map[string]interface{}{
			"model": "llama-3.2-11b-vision-preview",
			"messages": []map[string]string{
				{"role": "user", "content": "ping"},
			},
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("Groq test request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("Groq auth failed (HTTP %d): %s", resp.StatusCode, string(b))
		}

	default:
		// Gemini
		model := cfg.Model
		if model == "" {
			model = "gemini-2.0-flash"
		}
		endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
		reqBody := map[string]interface{}{
			"contents": []map[string]interface{}{
				{"parts": []map[string]string{{"text": "ping"}}},
			},
		}
		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := client.Post(endpoint, "application/json", bytes.NewBuffer(bodyBytes))
		if err != nil {
			return fmt.Errorf("Gemini test request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("Gemini auth failed (HTTP %d): %s", resp.StatusCode, string(b))
		}
	}

	log.Println("[Standalone AI] Credentials verified successfully! API Ping test passed.")
	return nil
}
