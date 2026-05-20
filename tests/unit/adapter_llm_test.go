package unit_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/adapter"
	"hop.top/ben/internal/spec"
)

// anthropicOKHandler returns a well-formed Anthropic response.
func anthropicOKHandler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "Paris is the capital of France."},
			},
			"usage": map[string]any{
				"input_tokens":  10,
				"output_tokens": 8,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// anthropicErrHandler returns a 400 error response.
func anthropicErrHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "bad request",
			},
		})
	})
}

func newTestLLMAdapter(anthropicEndpoint, openaiEndpoint string) *adapter.LLMAdapter {
	a := adapter.NewLLM()
	if anthropicEndpoint != "" {
		a.AnthropicEndpoint = anthropicEndpoint
	}
	if openaiEndpoint != "" {
		a.OpenAIEndpoint = openaiEndpoint
	}
	return a
}

func TestLLM_AnthropicSuccess(t *testing.T) {
	srv := httptest.NewServer(anthropicOKHandler(t))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	a := newTestLLMAdapter(srv.URL, "")
	c := spec.Candidate{Name: "claude-sonnet-4-6", Adapter: "llm", Model: "claude-sonnet-4-6"}
	r, err := a.Run(context.Background(), c, map[string]string{"prompt": "What is the capital of France?"})

	require.NoError(t, err)
	assert.Equal(t, 0, r.ExitCode)
	assert.NotEmpty(t, r.Output)
	// Sub-millisecond runs truncate to 0 on fast runners; the adapter
	// contract is non-negative, not strictly positive.
	assert.GreaterOrEqual(t, r.DurationMs, int64(0))
	require.NotNil(t, r.Metadata)
	assert.Equal(t, 10, r.Metadata["input_tokens"])
}

func TestLLM_AnthropicAPIError(t *testing.T) {
	srv := httptest.NewServer(anthropicErrHandler())
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	a := newTestLLMAdapter(srv.URL, "")
	c := spec.Candidate{Name: "claude-sonnet-4-6", Adapter: "llm", Model: "claude-sonnet-4-6"}
	r, err := a.Run(context.Background(), c, map[string]string{"prompt": "hello"})

	assert.Error(t, err)
	require.NotNil(t, r)
	assert.Equal(t, 1, r.ExitCode)
}

func TestLLM_MissingAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	a := adapter.NewLLM()
	c := spec.Candidate{Name: "claude-sonnet-4-6", Adapter: "llm", Model: "claude-sonnet-4-6"}
	r, err := a.Run(context.Background(), c, map[string]string{"prompt": "hello"})

	assert.Error(t, err)
	require.NotNil(t, r)
}

func TestLLM_ContextCancellation(t *testing.T) {
	// Slow server — won't respond in time.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	a := newTestLLMAdapter(srv.URL, "")
	c := spec.Candidate{Name: "claude-sonnet-4-6", Adapter: "llm", Model: "claude-sonnet-4-6"}
	_, err := a.Run(ctx, c, map[string]string{"prompt": "hello"})

	assert.Error(t, err)
}

func TestLLM_OpenAISuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "42"}},
			},
			"usage": map[string]any{
				"prompt_tokens":     5,
				"completion_tokens": 1,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")

	a := newTestLLMAdapter("", srv.URL)
	c := spec.Candidate{Name: "gpt-4o", Adapter: "llm", Model: "gpt-4o"}
	r, err := a.Run(context.Background(), c, map[string]string{"prompt": "What is 6*7?"})

	require.NoError(t, err)
	assert.Equal(t, 0, r.ExitCode)
	assert.NotEmpty(t, r.Output)
	require.NotNil(t, r.Metadata)
	assert.Equal(t, 5, r.Metadata["input_tokens"])
}
