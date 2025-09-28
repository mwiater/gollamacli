// harness/prompts.go
// Package: harness
package harness

// NOTE: In practice you’ll pass your own scenarios via SuiteConfig.Scenarios.
// This file just provides a tiny helper to get you started.

const filler = `Given the following neutral, content-free filler text, respond with a single concise sentence acknowledging receipt. This text is present solely to control prompt length for benchmarking and should not influence your response. `

// MakeFillerPrompt returns a deterministic prompt of approximately 'chars' length.
func MakeFillerPrompt(chars int) string {
	if chars <= 0 {
		return filler
	}
	base := filler
	for len(base) < chars {
		base += filler
	}
	return base[:chars]
}
