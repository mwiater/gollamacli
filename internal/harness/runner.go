// harness/runner.go
// Package: harness
package harness

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// RunSpeedSuite is the single exported entrypoint.
// Provide a fully-populated SuiteConfig, and it returns detailed results.
func RunSpeedSuite(ctx context.Context, cfg HarnessSuiteConfig) (HarnessSuiteResult, error) {
	if cfg.BaseURL == "" {
		return HarnessSuiteResult{}, errors.New("BaseURL is required (e.g., http://localhost:11434)")
	}
	if len(cfg.Models) == 0 {
		return HarnessSuiteResult{}, errors.New("at least one ModelConfig is required")
	}
	if len(cfg.Scenarios) == 0 {
		return HarnessSuiteResult{}, errors.New("at least one PromptScenario is required")
	}
	if cfg.Trials <= 0 {
		cfg.Trials = 5
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 60 * time.Second
	}

	client := newHTTPClient(cfg.RequestTimeout)
	var all []HarnessTrialResult

	for _, model := range cfg.Models {
		// Optional warm-up (not recorded)
		fmt.Println("Warming up:", model)
		if cfg.Warmup {
			_ = doWarmup(ctx, client, cfg.BaseURL, model, cfg.Scenarios[0])
		}

		// Optional single cold trial (tagged Cold=true)
		if cfg.IncludeCold {
			tr, err := GenerateAndMeasure(ctx, client, cfg.BaseURL, model, cfg.Scenarios[0], true)
			if err == nil {
				all = append(all, tr)
			}
			// Deliberately ignore cold errors to avoid aborting the whole suite.
		}

		// Warm trials across all scenarios
		for _, sc := range cfg.Scenarios {
			for i := 0; i < cfg.Trials; i++ {
				fmt.Println("GenerateAndMeasure:", sc)
				tr, err := GenerateAndMeasure(ctx, client, cfg.BaseURL, model, sc, false)
				if err != nil {
					// Record a synthetic failed row to make issues visible without aborting.
					all = append(all, HarnessTrialResult{
						ModelName:      model.Name,
						ScenarioID:     sc.ID,
						Cold:           false,
						PromptLenChars: len(sc.Prompt),
						TTFTMillis:     0,
						TotalMillis:    0,
						DoneReason:     fmt.Sprintf("error: %v", err),
					})
					continue
				}
				all = append(all, tr)
			}
		}
	}

	return buildHarnessSuiteResult(cfg, all), nil
}

func doWarmup(ctx context.Context, c *http.Client, base string, model HarnessModelConfig, scenario HarnessPromptScenario) error {
	ctx2, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	_, err := GenerateAndMeasure(ctx2, c, base, model, scenario, false)
	return err
}
