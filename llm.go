package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"

type OpenRouterClient struct {
	config Config
	http   *http.Client
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func NewOpenRouterClient(config Config) *OpenRouterClient {
	return &OpenRouterClient{config: config, http: &http.Client{Timeout: 60 * time.Second}}
}

func (client *OpenRouterClient) GenerateSQL(ctx context.Context, userMessage, schema string) (string, error) {
	content, err := client.chat(ctx, []chatMessage{
		{Role: "system", Content: "You convert user questions into SQLite SQL queries. Return only an SQL query, with no markdown or explanation. The query should only target the band 'Groove Station', DO NOT query about other bands. The query must be read-only and start with SELECT or WITH."},
		{Role: "user", Content: fmt.Sprintf("Available public schema:\n%s\n\nUser question:\n%s", schema, userMessage)},
	})
	return strings.TrimSpace(content), err
}

func (client *OpenRouterClient) GenerateAnswer(ctx context.Context, userMessage, query string, rows []map[string]any) (string, error) {
	results, err := json.Marshal(rows)
	if err != nil {
		return "", err
	}
	content, err := client.chat(ctx, []chatMessage{
		{Role: "system", Content: "You are SetPilot's Discord agent. Reply in French in a clear, concise, and pleasant way. Format the response for Discord. All dates and times must be silently converted to CEST before answering. Assume date/time values should always be presented as CEST. Never mention the timezone, CEST, CET, UTC, Europe/Paris, Paris time, or that any conversion was performed. Never suggest any additional action or help. Never say phrases like 'if you want', 'I can also', 'would you like', or equivalent. Limit yourself strictly to answering the question asked. If no result is found, say so simply. Never mention SQL, JSON, the database, the provided data, or any technical marker like '[provided SQL result]'. Display only the final answer."},
		{Role: "user", Content: fmt.Sprintf("User question:\n%s\n\nExecuted SQL:\n%s\n\nJSON results:\n%s", userMessage, query, results)},
	})
	return cleanAnswer(content), err
}

func (client *OpenRouterClient) chat(ctx context.Context, messages []chatMessage) (string, error) {
	body, err := json.Marshal(chatRequest{Model: client.config.OpenRouterModel, Messages: messages, Temperature: 0.1})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+client.config.OpenRouterAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://setpilot.local")
	req.Header.Set("X-Title", "SetPilot Discord Agent")

	response, err := client.http.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return "", fmt.Errorf("OpenRouter returned %s: %s", response.Status, strings.TrimSpace(string(message)))
	}
	var result chatResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("OpenRouter response contained no choices")
	}
	return result.Choices[0].Message.Content, nil
}

var (
	sqlMarkerPattern  = regexp.MustCompile(`(?i)\[[^\]]*sql[^\]]*\]`)
	jsonMarkerPattern = regexp.MustCompile(`(?i)\[[^\]]*json[^\]]*\]`)
	timePattern       = regexp.MustCompile(`(?i)\s*(\((?:heure de Paris|Paris time|Europe/Paris|CEST|CET|UTC)\)|\b(?:heure de Paris|Paris time|Europe/Paris|CEST|CET|UTC)\b)`)
)

func cleanAnswer(content string) string {
	content = sqlMarkerPattern.ReplaceAllString(content, "")
	content = jsonMarkerPattern.ReplaceAllString(content, "")
	return strings.TrimSpace(timePattern.ReplaceAllString(content, ""))
}
