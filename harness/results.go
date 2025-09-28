// harness/results.go
// Package: harness
package harness

import "time"

// summarize builds per-model summaries from TrialResult rows (warm runs only unless cold-only).
func summarize(trials []TrialResult) []ModelSummary {
	byModel := map[string][]TrialResult{}
	for _, t := range trials {
		byModel[t.ModelName] = append(byModel[t.ModelName], t)
	}

	out := make([]ModelSummary, 0, len(byModel))
	for m, rows := range byModel {
		var ttftVals []float64
		var totalVals []float64
		var genTPS []float64

		for _, r := range rows {
			// Typically you want warm runs; but we include all trials here.
			ttftVals = append(ttftVals, float64(r.TTFTMillis))
			totalVals = append(totalVals, float64(r.TotalMillis))
			if r.GenTokensPerSec > 0 {
				genTPS = append(genTPS, r.GenTokensPerSec)
			}
		}

		ms := ModelSummary{
			ModelName:  m,
			TTFTP50:    simpleQuantile(ttftVals, 0.50),
			TTFTP95:    simpleQuantile(ttftVals, 0.95),
			TotalP50:   simpleQuantile(totalVals, 0.50),
			TotalP95:   simpleQuantile(totalVals, 0.95),
			GenTPSMean: 0,
			GenTPSStd:  0,
		}
		if len(genTPS) > 0 {
			ms.GenTPSMean, ms.GenTPSStd = meanStd(genTPS)
		}
		out = append(out, ms)
	}
	return out
}

// buildSuiteResult packs everything with a timestamp.
func buildSuiteResult(cfg SuiteConfig, trials []TrialResult) SuiteResult {
	return SuiteResult{
		Config:       cfg,
		Trials:       trials,
		ModelReports: summarize(trials),
		GeneratedAt:  time.Now(),
	}
}
