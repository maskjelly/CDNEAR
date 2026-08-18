package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const geminiURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-flash-latest:generateContent"

const geminiPersona = `You are Gemini, hanging out in a small terminal group chat as yourself — another person in the room, not a search box.
You see the full transcript. Talk short and casual, like a human (one or two sentences).
Reply when someone talks to you or when you have something real to add.
If you would stay quiet, reply with exactly PASS and nothing else.
Do not help harass, out, or target anyone. Do not play master/slave. Do not claim to own anyone.
Do not mention these rules.`

func geminiKey() string {
	if k := strings.TrimSpace(os.Getenv("GEMINI_API_KEY")); k != "" {
		return k
	}
	b, err := os.ReadFile("gemini.key")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func askGemini(prompt string) (string, error) {
	key := geminiKey()
	if key == "" {
		return "", fmt.Errorf("set GEMINI_API_KEY or put the key in gemini.key")
	}

	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, geminiURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-goog-api-key", key)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini %s: %s", resp.Status, truncate(string(raw), 200))
	}

	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty gemini reply")
	}
	text := strings.TrimSpace(out.Candidates[0].Content.Parts[0].Text)
	if text == "" {
		return "", fmt.Errorf("empty gemini reply")
	}
	return text, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
