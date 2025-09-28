// harness/metrics.go
// Package: harness
package harness

import (
	"math"
	"slices"
)

// simpleQuantile returns the q-quantile (0..1) of a slice (copy-safe).
func simpleQuantile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	cp := slices.Clone(values)
	slices.Sort(cp)
	if q <= 0 {
		return cp[0]
	}
	if q >= 1 {
		return cp[len(cp)-1]
	}
	pos := q * float64(len(cp)-1)
	l := int(math.Floor(pos))
	r := int(math.Ceil(pos))
	if l == r {
		return cp[l]
	}
	frac := pos - float64(l)
	return cp[l]*(1-frac) + cp[r]*frac
}

func meanStd(values []float64) (mean, std float64) {
	n := float64(len(values))
	if n == 0 {
		return 0, 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean = sum / n
	var varsum float64
	for _, v := range values {
		d := v - mean
		varsum += d * d
	}
	std = math.Sqrt(varsum / n)
	return
}
