package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mwiater/gollamacli/harness"
)

func main() {
	cfg := harness.SuiteConfig{
		BaseURL: "http://localhost:11434",
		Models: []harness.ModelConfig{
			{
				Name: "stablelm-zephyr:3b", // example; use your sub-2B models
				Options: map[string]any{
					"temperature": 0.0,
					"top_p":       1.0,
					"top_k":       1,
					"num_predict": 256,              // cap output length (a must for fairness)
					"stop":        []string{"\n\n"}, // trivial stop to help normalize output length
				},
			},
			{
				Name: "gemma3n:e2b",
				Options: map[string]any{
					"temperature": 0.0,
					"top_p":       1.0,
					"top_k":       1,
					"num_predict": 256,
					"stop":        []string{"\n\n"},
				},
			},
		},
		Scenarios: []harness.PromptScenario{
			{ID: "short", Description: "≈512 chars", Prompt: harness.MakeFillerPrompt(512)},
			{ID: "medium", Description: "≈2048 chars", Prompt: harness.MakeFillerPrompt(2048)},
			{ID: "long", Description: "≈4096 chars", Prompt: harness.MakeFillerPrompt(4096)},
		},
		Trials:         5,
		Warmup:         true,
		IncludeCold:    true,
		RequestTimeout: 60 * time.Second,
	}

	res, err := harness.RunSpeedSuite(context.Background(), cfg)
	if err != nil {
		panic(err)
	}

	// Print a concise summary
	for _, m := range res.ModelReports {
		fmt.Printf("MODEL: %s\n", m.ModelName)
		fmt.Printf("  TTFT  p50/p95: %.1f / %.1f ms\n", m.TTFTP50, m.TTFTP95)
		fmt.Printf("  TOTAL p50/p95: %.1f / %.1f ms\n", m.TotalP50, m.TotalP95)
		fmt.Printf("  Gen TPS mean±std: %.2f ± %.2f tok/s\n\n", m.GenTPSMean, m.GenTPSStd)
	}

	// Or persist to JSON
	b, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(b))
}
