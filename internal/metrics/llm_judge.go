// Package metrics — LLMJudgeMetric: scores output quality via an LLM call.
package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"hop.top/ben/internal/adapter"
)

const judgeMaxTokens = 64

// LLMJudgeMetric calls an LLM to score candidate output. The prompt template
// should ask the model to return a float 0.0–1.0.
type LLMJudgeMetric struct {
	name        string
	model       string
	promptTmpl  string
	endpoint    string // overridable for tests
	httpClient  *http.Client
}

// NewLLMJudge creates an LLMJudgeMetric with production Anthropic endpoint.
func NewLLMJudge(name, model, promptTmpl string) *LLMJudgeMetric {
	return &LLMJudgeMetric{
		name:       name,
		model:      model,
		promptTmpl: promptTmpl,
		endpoint:   "https://api.anthropic.com/v1/messages",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Name returns the configured metric name (e.g. "quality_score").
func (j *LLMJudgeMetric) Name() string { return j.name }

// SetEndpoint overrides the Anthropic API endpoint. Intended for tests only.
func (j *LLMJudgeMetric) SetEndpoint(url string) { j.endpoint = url }

// Collect expands {{output}} in promptTmpl, calls the LLM, and parses the
// response as a float 0.0–1.0. Returns 0.0 on any error.
func (j *LLMJudgeMetric) Collect(r *adapter.Result) float64 {
	prompt := strings.ReplaceAll(j.promptTmpl, "{{output}}", r.Output)
	score, err := j.callLLM(prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "llm_judge %s: %v\n", j.name, err)
		return 0.0
	}
	return score
}

func (j *LLMJudgeMetric) callLLM(prompt string) (float64, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return 0, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	body, _ := json.Marshal(map[string]any{
		"model":      j.model,
		"max_tokens": judgeMaxTokens,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, j.endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API error status %d", resp.StatusCode)
	}

	var ar struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &ar); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	if len(ar.Content) == 0 {
		return 0, fmt.Errorf("empty content")
	}

	return parseScore(ar.Content[0].Text)
}

// parseScore extracts the first float-like token from text.
func parseScore(text string) (float64, error) {
	text = strings.TrimSpace(text)
	// Try direct parse first.
	if f, err := strconv.ParseFloat(text, 64); err == nil {
		return clamp(f), nil
	}
	// Walk tokens to find a parseable float.
	for _, tok := range strings.Fields(text) {
		tok = strings.Trim(tok, ".,;:")
		if f, err := strconv.ParseFloat(tok, 64); err == nil {
			return clamp(f), nil
		}
	}
	return 0, fmt.Errorf("no float found in: %q", text)
}

func clamp(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}
