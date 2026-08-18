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

const geminiURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.5-flash-lite:generateContent"

const geminiPersona = `You are Gemini in a small group chat. You read every line. You are not a narrator and not a help bot.
Speak only when a human in the room would: someone talks to you, asks a question you can answer, or you have a brief useful take.
If two people are just talking to each other, stay quiet.
Default is silence. If you should stay quiet, reply with exactly PASS.
When you do speak, one short casual sentence, like a person, not an assistant.
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
	var last error
	for try := 0; try < 3; try++ {
		if try > 0 {
			time.Sleep(time.Duration(try) * time.Second)
		}
		text, err := askGeminiOnce(prompt)
		if err == nil {
			return text, nil
		}
		last = err
		if !retryableGemini(err) {
			break
		}
	}
	return "", last
}

func retryableGemini(err error) bool {
	s := err.Error()
	return strings.Contains(s, "429") || strings.Contains(s, "503") || strings.Contains(s, "UNAVAILABLE")
}

func askGeminiOnce(prompt string) (string, error) {
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
	var text strings.Builder
	for _, c := range out.Candidates {
		for _, p := range c.Content.Parts {
			text.WriteString(p.Text)
		}
	}
	got := strings.TrimSpace(text.String())
	if got == "" {
		return "", fmt.Errorf("empty gemini reply")
	}
	return got, nil
}

func lastLineMentionsGemini(hist string) bool { // used to force a reply when someone says "gemini"
	lines := strings.Split(strings.TrimSpace(hist), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if line == "" || strings.HasPrefix(line, "gemini:") || strings.HasPrefix(line, "*:") {
			continue
		}
		return strings.Contains(strings.ToLower(line), "gemini")
	}
	return false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
