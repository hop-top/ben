// Package adapter — LLM adapter: Anthropic + OpenAI via net/http (no SDK).
package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"hop.top/ben/internal/spec"
)

const (
	anthropicURL = "https://api.anthropic.com/v1/messages"
	openaiURL    = "https://api.openai.com/v1/chat/completions"
	maxTokens    = 4096
)

// LLMAdapter implements Adapter for LLM candidates.
// Provider is detected from Candidate.Model:
//   - "claude-*" → Anthropic Messages API
//   - "gpt-*"    → OpenAI Chat Completions API
type LLMAdapter struct {
	// AnthropicEndpoint and OpenAIEndpoint are overridable for tests.
	AnthropicEndpoint string
	OpenAIEndpoint    string
	httpClient        *http.Client
}

// NewLLM creates an LLMAdapter with production endpoints.
func NewLLM() *LLMAdapter {
	return &LLMAdapter{
		AnthropicEndpoint: anthropicURL,
		OpenAIEndpoint:    openaiURL,
		httpClient:        &http.Client{Timeout: 120 * time.Second},
	}
}

// Run executes the LLM call for candidate c and returns a Result.
func (a *LLMAdapter) Run(ctx context.Context, c spec.Candidate, input map[string]string) (*Result, error) {
	model := c.Model
	if model == "" {
		model = c.Name // fallback: use candidate name as model
	}

	prompt := spec.Template(c.Cmd, input)
	if prompt == "" {
		// No cmd override — use task prompt from input map directly.
		prompt = input["prompt"]
	}

	start := time.Now()
	var (
		output      string
		inputToks   int
		outputToks  int
		apiErr      error
	)

	switch {
	case strings.HasPrefix(model, "claude"):
		output, inputToks, outputToks, apiErr = a.callAnthropic(ctx, model, prompt)
	case strings.HasPrefix(model, "gpt"):
		output, inputToks, outputToks, apiErr = a.callOpenAI(ctx, model, prompt)
	default:
		return &Result{ExitCode: 1, Err: fmt.Errorf("llm: unknown model provider for %q", model)}, fmt.Errorf("llm: unknown model provider for %q", model)
	}

	dur := time.Since(start).Milliseconds()

	if apiErr != nil {
		return &Result{
			ExitCode:   1,
			Err:        apiErr,
			DurationMs: dur,
			Metadata:   map[string]any{"model": model},
		}, apiErr
	}

	return &Result{
		Output:     output,
		ExitCode:   0,
		DurationMs: dur,
		Metadata: map[string]any{
			"model":         model,
			"input_tokens":  inputToks,
			"output_tokens": outputToks,
		},
	}, nil
}

// ---- Anthropic ----

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (a *LLMAdapter) callAnthropic(ctx context.Context, model, prompt string) (string, int, int, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", 0, 0, fmt.Errorf("llm: ANTHROPIC_API_KEY not set")
	}

	body, _ := json.Marshal(anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  []anthropicMessage{{Role: "user", Content: prompt}},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.AnthropicEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", 0, 0, fmt.Errorf("llm: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", 0, 0, fmt.Errorf("llm: anthropic http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, 0, fmt.Errorf("llm: read body: %w", err)
	}

	var ar anthropicResponse
	if err := json.Unmarshal(raw, &ar); err != nil {
		return "", 0, 0, fmt.Errorf("llm: decode anthropic response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("status %d", resp.StatusCode)
		if ar.Error != nil {
			msg = ar.Error.Message
		}
		return "", 0, 0, fmt.Errorf("llm: anthropic API error: %s", msg)
	}

	if len(ar.Content) == 0 {
		return "", 0, 0, fmt.Errorf("llm: anthropic: empty content")
	}
	return ar.Content[0].Text, ar.Usage.InputTokens, ar.Usage.OutputTokens, nil
}

// ---- OpenAI ----

type openAIRequest struct {
	Model     string          `json:"model"`
	Messages  []openAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (a *LLMAdapter) callOpenAI(ctx context.Context, model, prompt string) (string, int, int, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", 0, 0, fmt.Errorf("llm: OPENAI_API_KEY not set")
	}

	body, _ := json.Marshal(openAIRequest{
		Model:     model,
		Messages:  []openAIMessage{{Role: "user", Content: prompt}},
		MaxTokens: maxTokens,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.OpenAIEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", 0, 0, fmt.Errorf("llm: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", 0, 0, fmt.Errorf("llm: openai http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, 0, fmt.Errorf("llm: read body: %w", err)
	}

	var or openAIResponse
	if err := json.Unmarshal(raw, &or); err != nil {
		return "", 0, 0, fmt.Errorf("llm: decode openai response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("status %d", resp.StatusCode)
		if or.Error != nil {
			msg = or.Error.Message
		}
		return "", 0, 0, fmt.Errorf("llm: openai API error: %s", msg)
	}

	if len(or.Choices) == 0 {
		return "", 0, 0, fmt.Errorf("llm: openai: empty choices")
	}
	return or.Choices[0].Message.Content, or.Usage.PromptTokens, or.Usage.CompletionTokens, nil
}
