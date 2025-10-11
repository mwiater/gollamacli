// harness/ollama_client.go
// Package: harness
package harness

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type ollamaRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  bool           `json:"stream"`
	Options map[string]any `json:"options,omitempty"`
}

type ollamaStreamEvent struct {
	// For all events
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"` // chunk text (may be empty on the final done=true event)
	Done      bool      `json:"done"`

	// On done=true
	DoneReason         string `json:"done_reason,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"` // ns
	LoadDuration       int64  `json:"load_duration,omitempty"`  // ns
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"` // ns
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"` // ns
}

// GenerateAndMeasure performs a single streamed /api/generate call and measures timings.
func GenerateAndMeasure(
	ctx context.Context,
	httpClient *http.Client,
	baseURL string,
	model HarnessModelConfig,
	scenario HarnessPromptScenario,
	isCold bool,
) (HarnessTrialResult, error) {
	reqPayload := ollamaRequest{
		Model:   model.Name,
		Prompt:  scenario.Prompt,
		Stream:  true,
		Options: model.Options,
	}
	body, _ := json.Marshal(reqPayload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return HarnessTrialResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	// T0: just before send
	t0 := time.Now()
	resp, err := httpClient.Do(req)
	if err != nil {
		return HarnessTrialResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return HarnessTrialResult{}, fmt.Errorf("ollama error: status=%d body=%s", resp.StatusCode, string(b))
	}

	var (
		gotFirstChunk bool
		tFirst        time.Time
		final         ollamaStreamEvent
	)

	reader := bufio.NewReader(resp.Body)
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			var ev ollamaStreamEvent
			if err := json.Unmarshal(line, &ev); err == nil {
				if !gotFirstChunk && len(ev.Response) > 0 {
					gotFirstChunk = true
					tFirst = time.Now()
				}
				if ev.Done {
					final = ev
					break
				}
			}
			// silently skip malformed lines; continue
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return HarnessTrialResult{}, readErr
		}
	}

	tEnd := time.Now()

	// Derive metrics
	ttft := tEnd.Sub(tEnd) // default 0
	if gotFirstChunk {
		ttft = tFirst.Sub(t0)
	}
	total := tEnd.Sub(t0)

	// Options: extract num_predict if present
	maxTokens := 0
	if model.Options != nil {
		if v, ok := model.Options["num_predict"]; ok {
			switch vv := v.(type) {
			case float64:
				maxTokens = int(vv)
			case int:
				maxTokens = vv
			}
		}
	}

	tr := HarnessTrialResult{
		ModelName:      model.Name,
		ScenarioID:     scenario.ID,
		Cold:           isCold,
		PromptLenChars: len(scenario.Prompt),
		MaxTokens:      maxTokens,
		TTFTMillis:     ttft.Milliseconds(),
		TotalMillis:    total.Milliseconds(),

		LoadMillis:        final.LoadDuration / 1_000_000,
		PromptEvalCount:   final.PromptEvalCount,
		PromptEvalMillis:  final.PromptEvalDuration / 1_000_000,
		GenEvalCount:      final.EvalCount,
		GenEvalMillis:     final.EvalDuration / 1_000_000,
		TotalServerMillis: final.TotalDuration / 1_000_000,

		DoneReason: final.DoneReason,
	}

	// Derived throughputs
	if tr.PromptEvalMillis > 0 && tr.PromptEvalCount > 0 {
		tr.PromptTokensPerSec = float64(tr.PromptEvalCount) / (float64(tr.PromptEvalMillis) / 1000.0)
	}
	if tr.GenEvalMillis > 0 && tr.GenEvalCount > 0 {
		tr.GenTokensPerSec = float64(tr.GenEvalCount) / (float64(tr.GenEvalMillis) / 1000.0)
	}

	return tr, nil
}

// newHTTPClient returns a tuned HTTP client with keep-alives (important for consistency).
func newHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2: true,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}
