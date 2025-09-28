// harness/types.go
// Package: harness
package harness

import "time"

// ModelConfig defines how to call a specific model via Ollama.
type ModelConfig struct {
	Name        string         `json:"name"`         // e.g. "granite3.3:2b"
	DisplayName string         `json:"display_name"` // optional pretty name
	Options     map[string]any `json:"options"`      // Ollama /api/generate "options" (temperature, top_p, top_k, num_predict, stop, etc.)
}

// PromptScenario is one canonical test prompt with a human label.
type PromptScenario struct {
	ID          string `json:"id"`          // e.g. "short", "medium", "long"
	Description string `json:"description"` // human-friendly
	Prompt      string `json:"prompt"`      // full prompt text
}

// SuiteConfig configures the entire run.
type SuiteConfig struct {
	// Ollama endpoint like "http://localhost:11434"
	BaseURL string `json:"base_url"`

	// Models to benchmark.
	Models []ModelConfig `json:"models"`

	// Scenarios (short/medium/long, etc).
	Scenarios []PromptScenario `json:"scenarios"`

	// Per-scenario number of trials to run per model (warm runs).
	Trials int `json:"trials"`

	// Whether to run an initial warm-up request per model (not recorded).
	Warmup bool `json:"warmup"`

	// Whether to attempt a "cold" run before warm trials (unload/restart is user-managed).
	// If true, we tag the first run as Cold=true in TrialResult. Otherwise all are warm.
	IncludeCold bool `json:"include_cold"`

	// HTTP timeout per request (safety guard).
	RequestTimeout time.Duration `json:"request_timeout"`
}

// TrialResult captures metrics for a single streamed generation trial.
type TrialResult struct {
	ModelName      string `json:"model_name"`
	ScenarioID     string `json:"scenario_id"`
	Cold           bool   `json:"cold"` // true if this was the initial cold run
	PromptLenChars int    `json:"prompt_len_chars"`
	MaxTokens      int    `json:"max_tokens"` // extracted from options if set (num_predict)

	// Client timings (monotonic)
	TTFTMillis  int64 `json:"ttft_ms"`  // time-to-first-token
	TotalMillis int64 `json:"total_ms"` // end-to-end time

	// Server-reported (from final Ollama event)
	LoadMillis        int64 `json:"load_ms"`
	PromptEvalCount   int   `json:"prompt_eval_count"`
	PromptEvalMillis  int64 `json:"prompt_eval_ms"`
	GenEvalCount      int   `json:"eval_count"`
	GenEvalMillis     int64 `json:"eval_ms"`
	TotalServerMillis int64 `json:"total_server_ms"`

	// Derived rates
	PromptTokensPerSec float64 `json:"prompt_tokens_per_sec"`
	GenTokensPerSec    float64 `json:"gen_tokens_per_sec"`

	// Raw final event for debugging (optional)
	DoneReason string `json:"done_reason,omitempty"`
}

// ModelSummary aggregates per-model stats for reporting.
type ModelSummary struct {
	ModelName string `json:"model_name"`

	// p50/p95 for TTFT and Total latency across all warm trials
	TTFTP50  float64 `json:"ttft_p50_ms"`
	TTFTP95  float64 `json:"ttft_p95_ms"`
	TotalP50 float64 `json:"total_p50_ms"`
	TotalP95 float64 `json:"total_p95_ms"`

	// Mean +/- std for GenTokensPerSec
	GenTPSMean float64 `json:"gen_tps_mean"`
	GenTPSStd  float64 `json:"gen_tps_std"`
}

// SuiteResult is the top-level artifact returned by RunSpeedSuite.
type SuiteResult struct {
	Config       SuiteConfig    `json:"config"`
	Trials       []TrialResult  `json:"trials"`
	ModelReports []ModelSummary `json:"model_reports"`
	GeneratedAt  time.Time      `json:"generated_at"`
}
