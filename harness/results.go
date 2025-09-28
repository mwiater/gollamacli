// harness/results.go
// Package: harness
package harness

import "time"

// summarize builds per-model summaries from TrialResult rows (warm runs only unless cold-only).
func summarize(trials []HarnessTrialResult) []HarnessModelSummary {
	byModel := map[string][]HarnessTrialResult{}
	for _, t := range trials {
		byModel[t.ModelName] = append(byModel[t.ModelName], t)
	}

	out := make([]HarnessModelSummary, 0, len(byModel))
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

		ms := HarnessModelSummary{
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

// buildHarnessSuiteResult packs everything with a timestamp.
func buildHarnessSuiteResult(cfg HarnessSuiteConfig, trials []HarnessTrialResult) HarnessSuiteResult {
	return HarnessSuiteResult{
		Config:       cfg,
		Trials:       trials,
		ModelReports: summarize(trials),
		GeneratedAt:  time.Now(),
	}
}
