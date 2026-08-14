package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// SolveScreenshotDirect calls the configured AI Vision REST API directly with base64 screenshot data
func SolveScreenshotDirect(cfg *AIConfig, b64ImageData string) (string, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	apiKey = strings.ReplaceAll(apiKey, "\"", "")
	apiKey = strings.ReplaceAll(apiKey, "'", "")

	if apiKey == "" {
		return "", fmt.Errorf("AI API Key is empty in %s", GetAIConfigPath())
	}

	fullB64Url := b64ImageData
	if !strings.HasPrefix(fullB64Url, "data:") {
		fullB64Url = "data:image/jpeg;base64," + fullB64Url
	}

	prompt := cfg.CustomPrompt
	if prompt == "" {
		prompt = "Solve the coding question or problem shown in this screenshot. Output ONLY clean, working code without explanations or markdown formatting."
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var generatedText string

	switch cfg.Provider {
	case "openrouter":
		model := cfg.Model
		if model == "" {
			model = "openrouter/auto"
		}
		endpoint := "https://openrouter.ai/api/v1/chat/completions"

		maxTok := cfg.MaxTokens
		if maxTok <= 0 {
			maxTok = 2048
		}

		reqBody := map[string]interface{}{
			"model":      model,
			"max_tokens": maxTok,
			"messages": []map[string]interface{}{
				{
					"role": "user",
					"content": []map[string]interface{}{
						{"type": "text", "text": prompt},
						{"type": "image_url", "image_url": map[string]string{"url": fullB64Url}},
					},
				},
			},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("HTTP-Referer", "https://ctrlv.sync")
		req.Header.Set("X-Title", "ctrlv Standalone AI")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("OpenRouter API request error: %w", err)
		}
		defer resp.Body.Close()

		respBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("OpenRouter API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
		}

		var resStruct struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}

		if err := json.Unmarshal(respBytes, &resStruct); err != nil || len(resStruct.Choices) == 0 {
			return "", fmt.Errorf("failed to parse OpenRouter response: %s", string(respBytes))
		}

		generatedText = resStruct.Choices[0].Message.Content

	case "openai":
		model := cfg.Model
		if model == "" {
			model = "gpt-4o-mini"
		}
		endpoint := "https://api.openai.com/v1/chat/completions"

		maxTok := cfg.MaxTokens
		if maxTok <= 0 {
			maxTok = 2048
		}

		reqBody := map[string]interface{}{
			"model":      model,
			"max_tokens": maxTok,
			"messages": []map[string]interface{}{
				{
					"role": "user",
					"content": []map[string]interface{}{
						{"type": "text", "text": prompt},
						{"type": "image_url", "image_url": map[string]string{"url": fullB64Url}},
					},
				},
			},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("OpenAI API request error: %w", err)
		}
		defer resp.Body.Close()

		respBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("OpenAI API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
		}

		var resStruct struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}

		if err := json.Unmarshal(respBytes, &resStruct); err != nil || len(resStruct.Choices) == 0 {
			return "", fmt.Errorf("failed to parse OpenAI response: %s", string(respBytes))
		}

		generatedText = resStruct.Choices[0].Message.Content

	case "groq":
		model := cfg.Model
		if model == "" {
			model = "llama-3.2-11b-vision-preview"
		}
		endpoint := "https://api.groq.com/openai/v1/chat/completions"

		reqBody := map[string]interface{}{
			"model": model,
			"messages": []map[string]interface{}{
				{
					"role": "user",
					"content": []map[string]interface{}{
						{"type": "text", "text": prompt},
						{"type": "image_url", "image_url": map[string]string{"url": fullB64Url}},
					},
				},
			},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("Groq API request error: %w", err)
		}
		defer resp.Body.Close()

		respBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("Groq API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
		}

		var resStruct struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}

		if err := json.Unmarshal(respBytes, &resStruct); err != nil || len(resStruct.Choices) == 0 {
			return "", fmt.Errorf("failed to parse Groq response: %s", string(respBytes))
		}

		generatedText = resStruct.Choices[0].Message.Content

	default:
		// Gemini
		model := cfg.Model
		if model == "" {
			model = "gemini-2.0-flash"
		}
		endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

		cleanB64 := b64ImageData
		mimeType := "image/jpeg"
		if strings.HasPrefix(cleanB64, "data:") {
			parts := strings.Split(cleanB64, ";base64,")
			if len(parts) == 2 {
				mimeType = strings.ReplaceAll(parts[0], "data:", "")
				cleanB64 = parts[1]
			}
		}

		reqBody := map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]interface{}{
						{"text": prompt},
						{
							"inline_data": map[string]string{
								"mime_type": mimeType,
								"data":      cleanB64,
							},
						},
					},
				},
			},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := client.Post(endpoint, "application/json", bytes.NewBuffer(bodyBytes))
		if err != nil {
			return "", fmt.Errorf("Gemini API request error: %w", err)
		}
		defer resp.Body.Close()

		respBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("Gemini API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
		}

		var resStruct struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal(respBytes, &resStruct); err != nil || len(resStruct.Candidates) == 0 || len(resStruct.Candidates[0].Content.Parts) == 0 {
			return "", fmt.Errorf("failed to parse Gemini response: %s", string(respBytes))
		}

		generatedText = resStruct.Candidates[0].Content.Parts[0].Text
	}

	if cfg.CodeOnly {
		generatedText = ParseCleanCodeOnly(generatedText)
	}

	return generatedText, nil
}

// SolveTextDirect calls the configured AI REST API directly with clipboard question text (Pure Text API, No Image)
func SolveTextDirect(cfg *AIConfig, questionText string) (string, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	apiKey = strings.ReplaceAll(apiKey, "\"", "")
	apiKey = strings.ReplaceAll(apiKey, "'", "")

	if apiKey == "" {
		return "", fmt.Errorf("AI API Key is empty in %s", GetAIConfigPath())
	}

	userPrompt := cfg.CustomPrompt
	if userPrompt == "" {
		userPrompt = "Solve the coding question or problem shown in this text. Output ONLY clean, working code without explanations or markdown formatting."
	}

	fullPrompt := fmt.Sprintf("%s\n\nQuestion Context / Problem Statement:\n%s", userPrompt, questionText)

	client := &http.Client{Timeout: 30 * time.Second}
	var generatedText string

	switch cfg.Provider {
	case "openrouter":
		model := cfg.Model
		if model == "" {
			model = "openrouter/auto"
		}
		endpoint := "https://openrouter.ai/api/v1/chat/completions"

		maxTok := cfg.MaxTokens
		if maxTok <= 0 {
			maxTok = 2048
		}

		reqBody := map[string]interface{}{
			"model":      model,
			"max_tokens": maxTok,
			"messages": []map[string]string{
				{"role": "user", "content": fullPrompt},
			},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("HTTP-Referer", "https://ctrlv.sync")
		req.Header.Set("X-Title", "ctrlv Standalone AI")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("OpenRouter API request error: %w", err)
		}
		defer resp.Body.Close()

		respBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("OpenRouter API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
		}

		var resStruct struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}

		if err := json.Unmarshal(respBytes, &resStruct); err != nil || len(resStruct.Choices) == 0 {
			return "", fmt.Errorf("failed to parse OpenRouter response: %s", string(respBytes))
		}

		generatedText = resStruct.Choices[0].Message.Content

	case "openai":
		model := cfg.Model
		if model == "" {
			model = "gpt-4o-mini"
		}
		endpoint := "https://api.openai.com/v1/chat/completions"

		maxTok := cfg.MaxTokens
		if maxTok <= 0 {
			maxTok = 2048
		}

		reqBody := map[string]interface{}{
			"model":      model,
			"max_tokens": maxTok,
			"messages": []map[string]string{
				{"role": "user", "content": fullPrompt},
			},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("OpenAI API request error: %w", err)
		}
		defer resp.Body.Close()

		respBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("OpenAI API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
		}

		var resStruct struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}

		if err := json.Unmarshal(respBytes, &resStruct); err != nil || len(resStruct.Choices) == 0 {
			return "", fmt.Errorf("failed to parse OpenAI response: %s", string(respBytes))
		}

		generatedText = resStruct.Choices[0].Message.Content

	case "groq":
		model := cfg.Model
		if model == "" {
			model = "llama-3.2-11b-vision-preview"
		}
		endpoint := "https://api.groq.com/openai/v1/chat/completions"

		reqBody := map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{"role": "user", "content": fullPrompt},
			},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("Groq API request error: %w", err)
		}
		defer resp.Body.Close()

		respBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("Groq API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
		}

		var resStruct struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}

		if err := json.Unmarshal(respBytes, &resStruct); err != nil || len(resStruct.Choices) == 0 {
			return "", fmt.Errorf("failed to parse Groq response: %s", string(respBytes))
		}

		generatedText = resStruct.Choices[0].Message.Content

	default:
		// Gemini
		model := cfg.Model
		if model == "" {
			model = "gemini-2.0-flash"
		}
		endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

		reqBody := map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]string{
						{"text": fullPrompt},
					},
				},
			},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := client.Post(endpoint, "application/json", bytes.NewBuffer(bodyBytes))
		if err != nil {
			return "", fmt.Errorf("Gemini API request error: %w", err)
		}
		defer resp.Body.Close()

		respBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("Gemini API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
		}

		var resStruct struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal(respBytes, &resStruct); err != nil || len(resStruct.Candidates) == 0 || len(resStruct.Candidates[0].Content.Parts) == 0 {
			return "", fmt.Errorf("failed to parse Gemini response: %s", string(respBytes))
		}

		generatedText = resStruct.Candidates[0].Content.Parts[0].Text
	}

	if cfg.CodeOnly {
		generatedText = ParseCleanCodeOnly(generatedText)
	}

	return generatedText, nil
}

// ParseCleanCodeOnly strips markdown code fences if present
func ParseCleanCodeOnly(rawText string) string {
	if rawText == "" {
		return ""
	}

	re := regexp.MustCompile("(?s)```(?:[a-zA-Z0-9_+-]+)?\n(.*?)```")
	matches := re.FindAllStringSubmatch(rawText, -1)

	if len(matches) > 0 {
		var blocks []string
		for _, m := range matches {
			if len(m) > 1 {
				blocks = append(blocks, strings.TrimSpace(m[1]))
			}
		}
		return strings.Join(blocks, "\n\n")
	}

	return strings.TrimSpace(rawText)
}
