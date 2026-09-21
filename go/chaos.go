// Package chaosmeshlite runs chaos experiments as CI-runnable SLO assertions.
//
// It injects deterministic faults (added latency, periodic errors) into a
// stream of observed call latencies, then asserts that Service Level
// Objectives still hold. The fault schedule is deterministic (not RNG) so
// results are reproducible and identical across every language port.
package chaosmeshlite

import (
	"fmt"
	"math"
	"slices"
)

// FaultPolicy describes the faults injected into an experiment. The zero value
// (no added latency, FailEveryN == 0) injects nothing.
type FaultPolicy struct {
	// AddedLatencyMs is added to every non-failed call's latency.
	AddedLatencyMs float64
	// FailEveryN fails calls whose 1-indexed position is a multiple of N.
	// A value of 0 means no calls ever fail.
	FailEveryN int
}

// ExperimentResult is the measured outcome of a single experiment run.
type ExperimentResult struct {
	// TotalCalls is the number of observed calls, including failures.
	TotalCalls int
	// Failures is the number of injected failures.
	Failures int
	// SuccessRate is successes/total, rounded to six decimals (0 when empty).
	SuccessRate float64
	// P95LatencyMs is the 95th-percentile latency of successful calls.
	P95LatencyMs float64
}

// Slo is a Service Level Objective to assert against an ExperimentResult.
type Slo struct {
	// MinSuccessRate is the lowest acceptable success rate.
	MinSuccessRate float64
	// MaxP95LatencyMs is the highest acceptable p95 latency.
	MaxP95LatencyMs float64
}

// SloResult reports whether an experiment met its SLO and why not.
type SloResult struct {
	// Passed is true when there are no breaches.
	Passed bool
	// SuccessRate mirrors the evaluated ExperimentResult.
	SuccessRate float64
	// P95LatencyMs mirrors the evaluated ExperimentResult.
	P95LatencyMs float64
	// Breaches holds human-readable reasons the SLO failed (empty when passed).
	Breaches []string
}

// Percentile returns the nearest-rank percentile of values for p in [0,1].
// An empty slice yields 0. The input slice is not modified.
func Percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	ordered := slices.Clone(values)
	slices.Sort(ordered)
	rank := int(math.Ceil(p * float64(len(ordered))))
	idx := min(max(rank-1, 0), len(ordered)-1)
	return ordered[idx]
}

// RunExperiment applies policy to baseLatenciesMs and measures the outcome.
// Failed calls contribute no latency sample; every surviving call has
// AddedLatencyMs added to its base latency.
func RunExperiment(baseLatenciesMs []float64, policy FaultPolicy) ExperimentResult {
	latencies := make([]float64, 0, len(baseLatenciesMs))
	failures := 0
	for i, base := range baseLatenciesMs {
		oneIndexed := i + 1
		isFailure := policy.FailEveryN > 0 && oneIndexed%policy.FailEveryN == 0
		if isFailure {
			failures++
			continue
		}
		latencies = append(latencies, base+policy.AddedLatencyMs)
	}

	total := len(baseLatenciesMs)
	successes := total - failures
	successRate := 0.0
	if total != 0 {
		successRate = round6(float64(successes) / float64(total))
	}
	return ExperimentResult{
		TotalCalls:   total,
		Failures:     failures,
		SuccessRate:  successRate,
		P95LatencyMs: round6(Percentile(latencies, 0.95)),
	}
}

// EvaluateSlo checks result against slo and returns a pass/fail verdict with
// breach reasons. The success-rate breach (if any) is reported before the
// latency breach.
func EvaluateSlo(result ExperimentResult, slo Slo) SloResult {
	breaches := []string{}
	if result.SuccessRate < slo.MinSuccessRate {
		breaches = append(breaches,
			fmt.Sprintf("success_rate %.4f < %.4f", result.SuccessRate, slo.MinSuccessRate))
	}
	if result.P95LatencyMs > slo.MaxP95LatencyMs {
		breaches = append(breaches,
			fmt.Sprintf("p95 %.2fms > %.2fms", result.P95LatencyMs, slo.MaxP95LatencyMs))
	}
	return SloResult{
		Passed:       len(breaches) == 0,
		SuccessRate:  result.SuccessRate,
		P95LatencyMs: result.P95LatencyMs,
		Breaches:     breaches,
	}
}

func round6(v float64) float64 {
	return math.Round(v*1_000_000.0) / 1_000_000.0
}
