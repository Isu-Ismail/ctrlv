package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type AIConfig struct {
	Provider     string `json:"provider"`      // "openrouter", "groq", "google", "openai"
	APIKey       string `json:"api_key"`       // API key string
	Model        string `json:"model"`         // e.g. "openrouter/auto", "gemini-2.0-flash", "gpt-4o-mini", "llama-3.2-11b-vision-preview"
	CustomPrompt string `json:"custom_prompt"` // Instructions for vision AI
	MaxTokens    int    `json:"max_tokens"`    // Max output tokens limit
	CodeOnly     bool   `json:"code_only"`     // Strip markdown code fences
}

func GetAIConfigPath() string {
	return GetConfigPath()
}

func LoadAIConfig() (*AIConfig, error) {
	configPath, appCfg, err := EnsureConfigExists()
	if err != nil {
		return nil, err
	}

	aiCfg := appCfg.ToAIConfig()
	if strings.TrimSpace(aiCfg.APIKey) == "" {
		return aiCfg, fmt.Errorf("API Key is empty in %s. Run 'ctrlv config' to edit your config file", configPath)
	}

	return aiCfg, nil
}

func SaveAIConfig(cfg *AIConfig) error {
	_, appCfg, err := EnsureConfigExists()
	if err != nil {
		appCfg = &AppConfig{}
	}
	appCfg.Provider = cfg.Provider
	appCfg.APIKey = cfg.APIKey
	if cfg.Model != "" {
		appCfg.Model = cfg.Model
	}
	if cfg.CustomPrompt != "" {
		appCfg.CustomPrompt = cfg.CustomPrompt
	}
	if cfg.MaxTokens > 0 {
		appCfg.MaxTokens = cfg.MaxTokens
	}
	return SaveAppConfig(appCfg)
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
		maxTok := 50
		reqBody := map[string]interface{}{
			"model":      cfg.Model,
			"max_tokens": maxTok,
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

	case "openai":
		endpoint := "https://api.openai.com/v1/chat/completions"
		model := cfg.Model
		if model == "" {
			model = "gpt-4o-mini"
		}
		reqBody := map[string]interface{}{
			"model":      model,
			"max_tokens": 50,
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
			return fmt.Errorf("OpenAI test request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("OpenAI auth failed (HTTP %d): %s", resp.StatusCode, string(b))
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
