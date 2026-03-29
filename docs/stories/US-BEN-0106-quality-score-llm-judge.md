# US-BEN-0106 — Add quality_score metric via llm_judge for subjective output

**ID:** US-BEN-0106
**Title:** Add quality_score metric via llm_judge for subjective output evaluation
**Persona:** Platform Engineer
**Trigger:** Engineer benchmarks two tools with hard-to-diff outputs; wants an LLM judge to
score relevance objectively so `quality_score` feeds the weighted scorer.

---

## Acceptance Criteria

1. `quality_score` declared as `type: llm_judge` in config; ben loads it without code changes.
2. `ben run` with `--metric quality_score` invokes the judge once per candidate output.
3. Each candidate result includes `metrics.quality_score` as a float in [0, 1].
4. LLM judge prompt receives `{{output}}` substituted with actual candidate stdout.
5. Judge result is stored in the run JSON alongside other metrics.
6. If judge returns value outside [0, 1], ben clamps to range and logs a stderr warning.
7. `quality_score` participates in weighted scoring normally.

---

## Metrics Exercised

- `quality_score` (llm_judge plugin)
- `latency_ms` (builtin, co-present in run)

---

## Scorer Strategy

`weighted` — `latency_ms=0.3`, `quality_score=0.7`

---

## Happy Path Steps

```
1. Engineer adds to .ben/ben.yaml:
     plugins:
       metrics:
         - name: quality_score
           type: llm_judge
           model: claude-sonnet-4-6
           prompt: "Rate the relevance of this output 0-1: {{output}}"

2. Engineer runs:
     ben run --suite codebase-indexing \
       --metric latency_ms,quality_score \
       --scorer weighted:latency_ms=0.3,quality_score=0.7 \
       --format json

3. Ben executes both candidates; captures latency_ms.
4. For each candidate, ben calls llm_judge with candidate raw_output.
5. Judge returns float; stored as quality_score.
6. Scorer computes weighted score; assigns winner.
7. Result JSON shows quality_score per candidate.
```

---

## Failure Path + Expected Behavior

| Failure | Expected |
|---------|----------|
| LLM API returns non-numeric value | Exit 1; stderr: "quality_score: judge returned invalid value" |
| LLM API returns 1.5 (out of range) | Clamped to 1.0; stderr warning: "quality_score clamped" |
| `quality_score` declared but model not configured | Exit 1; stderr: "llm_judge: model is required" |
| LLM API timeout | Exit 1; stderr: "quality_score: judge timeout after Xs" |

---

## E2E Test Spec

**File:** `tests/e2e/stories/US_BEN_0106_test.go`
**Test func:** `TestUS_BEN_0106_QualityScoreLLMJudge`

Asserts:
- Config file declares `quality_score` as `type: llm_judge`.
- `ben run --metric latency_ms,quality_score --format json` exits 0.
- Each candidate in `candidates[]` has `metrics.quality_score` (float64).
- `metrics.quality_score` is in range [0.0, 1.0] for all candidates.
- `winner` is non-null (weighted scorer applied).
- When judge stub returns 1.5: result clamps to 1.0; stderr contains "clamped".
- When `model` omitted from config: exit 1; stderr contains "model is required".
