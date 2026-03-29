package unit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/adapter"
	"hop.top/ben/internal/metrics"
)

// ---- tokensMetric ----

func TestTokensMetric_Sum(t *testing.T) {
	m, ok := metrics.Get("tokens")
	require.True(t, ok, "tokens metric must be registered")

	r := &adapter.Result{
		Metadata: map[string]any{
			"input_tokens":  100,
			"output_tokens": 50,
		},
	}
	assert.Equal(t, float64(150), m.Collect(r))
}

func TestTokensMetric_MissingMetadata(t *testing.T) {
	m, ok := metrics.Get("tokens")
	require.True(t, ok)
	r := &adapter.Result{}
	assert.Equal(t, float64(0), m.Collect(r))
}

// ---- costUSDMetric ----

func TestCostUSD_Claude(t *testing.T) {
	m, ok := metrics.Get("cost_usd")
	require.True(t, ok)

	// claude-sonnet-4-6: $3/1M input, $15/1M output
	// 1000 input + 500 output = 0.003 + 0.0075 = 0.0105
	r := &adapter.Result{
		Metadata: map[string]any{
			"model":         "claude-sonnet-4-6",
			"input_tokens":  1000,
			"output_tokens": 500,
		},
	}
	cost := m.Collect(r)
	assert.InDelta(t, 0.0105, cost, 1e-9)
}

func TestCostUSD_GPT4o(t *testing.T) {
	m, ok := metrics.Get("cost_usd")
	require.True(t, ok)

	// gpt-4o: $2.50/1M input, $10/1M output
	// 2000 input + 1000 output = 0.005 + 0.01 = 0.015
	r := &adapter.Result{
		Metadata: map[string]any{
			"model":         "gpt-4o",
			"input_tokens":  2000,
			"output_tokens": 1000,
		},
	}
	cost := m.Collect(r)
	assert.InDelta(t, 0.015, cost, 1e-9)
}

func TestCostUSD_UnknownModel(t *testing.T) {
	m, ok := metrics.Get("cost_usd")
	require.True(t, ok)

	r := &adapter.Result{
		Metadata: map[string]any{
			"model":         "unknown-model-xyz",
			"input_tokens":  1000,
			"output_tokens": 500,
		},
	}
	assert.Equal(t, float64(0), m.Collect(r))
}

// ---- LLMJudgeMetric ----

func newJudgeServer(responseText string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": responseText},
			},
		})
	}))
}

func TestLLMJudge_Score(t *testing.T) {
	srv := newJudgeServer("0.85", http.StatusOK)
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	j := metrics.NewLLMJudge("quality_score", "claude-sonnet-4-6", "Rate 0-1: {{output}}")
	j.SetEndpoint(srv.URL) // test hook

	r := &adapter.Result{Output: "Paris is the capital of France."}
	score := j.Collect(r)
	assert.InDelta(t, 0.85, score, 1e-9)
}

func TestLLMJudge_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return non-JSON garbage.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json at all!!!"))
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	j := metrics.NewLLMJudge("quality_score", "claude-sonnet-4-6", "Rate 0-1: {{output}}")
	j.SetEndpoint(srv.URL)

	r := &adapter.Result{Output: "some output"}
	score := j.Collect(r) // must not panic
	assert.Equal(t, float64(0), score)
}

func TestLLMJudge_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	j := metrics.NewLLMJudge("quality_score", "claude-sonnet-4-6", "Rate 0-1: {{output}}")
	j.SetEndpoint(srv.URL)

	r := &adapter.Result{Output: "some output"}
	score := j.Collect(r)
	assert.Equal(t, float64(0), score)
}

func TestLLMJudge_TextWithFloat(t *testing.T) {
	srv := newJudgeServer("The score is 0.72 out of 1.0.", http.StatusOK)
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	j := metrics.NewLLMJudge("quality_score", "claude-sonnet-4-6", "Rate 0-1: {{output}}")
	j.SetEndpoint(srv.URL)

	r := &adapter.Result{Output: "some output"}
	score := j.Collect(r)
	assert.InDelta(t, 0.72, score, 1e-9)
}
