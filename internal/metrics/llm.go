// Package metrics — LLM-specific metrics: tokens and cost_usd.
package metrics

import "hop.top/ben/internal/adapter"

func init() {
	Register(tokensMetric{})
	Register(costUSDMetric{})
}

// modelPricing holds per-model price in USD per 1M tokens.
type modelPricing struct {
	inputPer1M  float64
	outputPer1M float64
}

// pricingTable maps model name to pricing.
// Add entries here as models become available.
var pricingTable = map[string]modelPricing{
	"claude-sonnet-4-6":    {inputPer1M: 3.00, outputPer1M: 15.00},
	"claude-3-5-sonnet":    {inputPer1M: 3.00, outputPer1M: 15.00},
	"claude-3-opus":        {inputPer1M: 15.00, outputPer1M: 75.00},
	"claude-3-haiku":       {inputPer1M: 0.25, outputPer1M: 1.25},
	"gpt-4o":               {inputPer1M: 2.50, outputPer1M: 10.00},
	"gpt-4o-mini":          {inputPer1M: 0.15, outputPer1M: 0.60},
	"gpt-4-turbo":          {inputPer1M: 10.00, outputPer1M: 30.00},
}

func metaInt(r *adapter.Result, key string) int {
	if r.Metadata == nil {
		return 0
	}
	v, ok := r.Metadata[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

func metaString(r *adapter.Result, key string) string {
	if r.Metadata == nil {
		return ""
	}
	v, ok := r.Metadata[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// tokensMetric returns total token count (input + output) from Metadata.
type tokensMetric struct{}

func (tokensMetric) Name() string { return "tokens" }
func (tokensMetric) Collect(r *adapter.Result) float64 {
	in := metaInt(r, "input_tokens")
	out := metaInt(r, "output_tokens")
	return float64(in + out)
}

// costUSDMetric computes cost in USD using pricingTable + token counts from Metadata.
type costUSDMetric struct{}

func (costUSDMetric) Name() string { return "cost_usd" }
func (costUSDMetric) Collect(r *adapter.Result) float64 {
	model := metaString(r, "model")
	pricing, ok := pricingTable[model]
	if !ok {
		return 0
	}
	in := float64(metaInt(r, "input_tokens"))
	out := float64(metaInt(r, "output_tokens"))
	return (in*pricing.inputPer1M + out*pricing.outputPer1M) / 1_000_000
}
