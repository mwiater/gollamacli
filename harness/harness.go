package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/k0kubun/pp"
)

//"stablelm-zephyr:3b",
//"granite3.3:2b",
//"gemma3n:e2b",
//"deepseek-r1:1.5b",
//"llama3.2:1b",
//"granite3.1-moe:1b",
//"dolphin-phi:2.7b",
//"qwen3:1.7b"

func Run() {
	cfg := HarnessSuiteConfig{
		BaseURL: "https://o-udoo01.0nezer0.com",
		Models: []HarnessModelConfig{
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
			{
				Name: "granite3.1-moe:1b",
				Options: map[string]any{
					"temperature": 0.0,
					"top_p":       1.0,
					"top_k":       1,
					"num_predict": 256,
					"stop":        []string{"\n\n"},
				},
			},
			{
				Name: "llama3.2:1b",
				Options: map[string]any{
					"temperature": 0.0,
					"top_p":       1.0,
					"top_k":       1,
					"num_predict": 256,
					"stop":        []string{"\n\n"},
				},
			},
		},
		Scenarios: []HarnessPromptScenario{
			{ID: "short", Description: "≈128 chars", Prompt: MakeFillerPrompt(128)},
			//{ID: "medium", Description: "≈2048 chars", Prompt: MakeFillerPrompt(2048)},
			//{ID: "long", Description: "≈4096 chars", Prompt: MakeFillerPrompt(4096)},
		},
		Trials:         3,
		Warmup:         true,
		IncludeCold:    true,
		RequestTimeout: 2 * time.Minute,
	}

	pp.Println(cfg)

	res, err := RunSpeedSuite(context.Background(), cfg)
	if err != nil {
		panic(err)
	}

	fmt.Println("???")

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
