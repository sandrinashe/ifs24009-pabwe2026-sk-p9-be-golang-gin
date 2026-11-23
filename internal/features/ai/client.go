package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ChatClient memanggil OpenAI-compatible Chat Completions.
type ChatClient struct {
	Endpoint string
	Key      string
	Model    string
	HTTP     *http.Client
}

// NewChatClient membuat client (timeout 90 detik).
func NewChatClient(endpoint, key, model string) *ChatClient {
	return &ChatClient{Endpoint: strings.TrimRight(endpoint, "/"), Key: key, Model: model,
		HTTP: &http.Client{Timeout: 90 * time.Second}}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Complete mengirim system+user prompt dan mengembalikan teks jawaban.
func (c *ChatClient) Complete(ctx context.Context, system, user string) (string, error) {
	if c.Endpoint == "" {
		return "", errors.New("OpenAI endpoint belum dikonfigurasi")
	}
	payload, err := json.Marshal(map[string]any{
		"model":       c.Model,
		"temperature": 0.7,
		"messages":    []chatMessage{{"system", system}, {"user", user}},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Key != "" {
		req.Header.Set("Authorization", "Bearer "+c.Key)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("OpenAI membalas status %d", res.StatusCode)
	}
	var parsed struct {
		Choices []struct {
			Message chatMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Choices) == 0 {
		return "", errors.New("respons OpenAI tidak dikenali")
	}
	return parsed.Choices[0].Message.Content, nil
}
