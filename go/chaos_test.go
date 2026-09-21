package chaosmeshlite

import (
	"strings"
	"testing"
)

func repeat(v float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = v
	}
	return out
}

func rangeFloats(lo, hi int) []float64 {
	out := make([]float64, 0, hi-lo+1)
	for i := lo; i <= hi; i++ {
		out = append(out, float64(i))
	}
	return out
}

func TestPercentileNearestRank(t *testing.T) {
	data := rangeFloats(1, 100)
	if got := Percentile(data, 0.95); got != 95.0 {
		t.Fatalf("p95 = %v, want 95", got)
	}
	if got := Percentile(data, 1.0); got != 100.0 {
		t.Fatalf("p100 = %v, want 100", got)
	}
	if got := Percentile([]float64{}, 0.95); got != 0.0 {
		t.Fatalf("empty = %v, want 0", got)
	}
}

func TestPercentileZeroTakesSmallest(t *testing.T) {
	if got := Percentile(rangeFloats(1, 100), 0.0); got != 1.0 {
		t.Fatalf("p0 = %v, want 1 (smallest, clamped rank)", got)
	}
}

func TestPercentileSingleElement(t *testing.T) {
	if got := Percentile([]float64{42.0}, 0.95); got != 42.0 {
		t.Fatalf("single = %v, want 42", got)
	}
}

func TestPercentileDoesNotMutateInput(t *testing.T) {
	data := []float64{3.0, 1.0, 2.0}
	_ = Percentile(data, 0.5)
	if data[0] != 3.0 || data[1] != 1.0 || data[2] != 2.0 {
		t.Fatalf("input was mutated: %v", data)
	}
}

func TestNoFaultAllSucceed(t *testing.T) {
	result := RunExperiment(repeat(100.0, 50), FaultPolicy{})
	if result.Failures != 0 {
		t.Fatalf("failures = %d, want 0", result.Failures)
	}
	if result.SuccessRate != 1.0 {
		t.Fatalf("success rate = %v, want 1", result.SuccessRate)
	}
	if result.P95LatencyMs != 100.0 {
		t.Fatalf("p95 = %v, want 100", result.P95LatencyMs)
	}
}

func TestLatencyInjectionRaisesP95(t *testing.T) {
	result := RunExperiment(repeat(100.0, 50), FaultPolicy{AddedLatencyMs: 250.0})
	if result.P95LatencyMs != 350.0 {
		t.Fatalf("p95 = %v, want 350", result.P95LatencyMs)
	}
}

func TestErrorInjectionLowersSuccessRate(t *testing.T) {
	result := RunExperiment(repeat(100.0, 100), FaultPolicy{FailEveryN: 10})
	if result.Failures != 10 {
		t.Fatalf("failures = %d, want 10", result.Failures)
	}
	if result.SuccessRate != 0.9 {
		t.Fatalf("success rate = %v, want 0.9", result.SuccessRate)
	}
}

func TestSloPassesWhenHealthy(t *testing.T) {
	result := RunExperiment(repeat(100.0, 100), FaultPolicy{})
	outcome := EvaluateSlo(result, Slo{MinSuccessRate: 0.99, MaxP95LatencyMs: 200.0})
	if !outcome.Passed {
		t.Fatalf("expected pass, got breaches %v", outcome.Breaches)
	}
	if len(outcome.Breaches) != 0 {
		t.Fatalf("expected no breaches, got %v", outcome.Breaches)
	}
}

func TestSloFailsOnLatencyBreach(t *testing.T) {
	result := RunExperiment(repeat(100.0, 100), FaultPolicy{AddedLatencyMs: 500.0})
	outcome := EvaluateSlo(result, Slo{MinSuccessRate: 0.99, MaxP95LatencyMs: 200.0})
	if outcome.Passed {
		t.Fatal("expected fail")
	}
	if !hasSubstring(outcome.Breaches, "p95") {
		t.Fatalf("expected a p95 breach, got %v", outcome.Breaches)
	}
}

func TestSloFailsOnErrorBreach(t *testing.T) {
	result := RunExperiment(repeat(100.0, 100), FaultPolicy{FailEveryN: 4})
	outcome := EvaluateSlo(result, Slo{MinSuccessRate: 0.99, MaxP95LatencyMs: 1000.0})
	if outcome.Passed {
		t.Fatal("expected fail")
	}
	if !hasSubstring(outcome.Breaches, "success_rate") {
		t.Fatalf("expected a success_rate breach, got %v", outcome.Breaches)
	}
}

func TestCombinedFaults(t *testing.T) {
	result := RunExperiment(repeat(50.0, 20), FaultPolicy{AddedLatencyMs: 100.0, FailEveryN: 5})
	if result.Failures != 4 {
		t.Fatalf("failures = %d, want 4 (calls 5,10,15,20)", result.Failures)
	}
	if result.P95LatencyMs != 150.0 {
		t.Fatalf("p95 = %v, want 150", result.P95LatencyMs)
	}
}

func TestEmptyBaseLatencies(t *testing.T) {
	result := RunExperiment([]float64{}, FaultPolicy{})
	if result.TotalCalls != 0 || result.Failures != 0 {
		t.Fatalf("totals = %d/%d, want 0/0", result.TotalCalls, result.Failures)
	}
	if result.SuccessRate != 0.0 {
		t.Fatalf("success rate = %v, want 0 for empty input", result.SuccessRate)
	}
	if result.P95LatencyMs != 0.0 {
		t.Fatalf("p95 = %v, want 0 for empty input", result.P95LatencyMs)
	}
}

func TestAllCallsFailLeaveNoLatencies(t *testing.T) {
	result := RunExperiment(repeat(100.0, 10), FaultPolicy{FailEveryN: 1})
	if result.Failures != 10 {
		t.Fatalf("failures = %d, want 10 (every call fails)", result.Failures)
	}
	if result.SuccessRate != 0.0 {
		t.Fatalf("success rate = %v, want 0", result.SuccessRate)
	}
	if result.P95LatencyMs != 0.0 {
		t.Fatalf("p95 = %v, want 0 when no successful calls", result.P95LatencyMs)
	}
}

func TestBothBreachesReportedSuccessRateFirst(t *testing.T) {
	result := RunExperiment(repeat(100.0, 100), FaultPolicy{AddedLatencyMs: 500.0, FailEveryN: 4})
	outcome := EvaluateSlo(result, Slo{MinSuccessRate: 0.99, MaxP95LatencyMs: 200.0})
	if outcome.Passed {
		t.Fatal("expected fail")
	}
	if len(outcome.Breaches) != 2 {
		t.Fatalf("expected 2 breaches, got %v", outcome.Breaches)
	}
	if !strings.Contains(outcome.Breaches[0], "success_rate") {
		t.Fatalf("expected success_rate breach first, got %v", outcome.Breaches)
	}
	if !strings.Contains(outcome.Breaches[1], "p95") {
		t.Fatalf("expected p95 breach second, got %v", outcome.Breaches)
	}
}

func TestBreachMessageFormat(t *testing.T) {
	result := RunExperiment(repeat(100.0, 100), FaultPolicy{AddedLatencyMs: 500.0})
	outcome := EvaluateSlo(result, Slo{MinSuccessRate: 0.99, MaxP95LatencyMs: 200.0})
	if outcome.Breaches[0] != "p95 600.00ms > 200.00ms" {
		t.Fatalf("breach = %q, want 'p95 600.00ms > 200.00ms'", outcome.Breaches[0])
	}
}

func TestFailEveryNZeroNeverFails(t *testing.T) {
	result := RunExperiment(repeat(10.0, 7), FaultPolicy{FailEveryN: 0})
	if result.Failures != 0 {
		t.Fatalf("failures = %d, want 0 when FailEveryN is 0", result.Failures)
	}
	if result.SuccessRate != 1.0 {
		t.Fatalf("success rate = %v, want 1", result.SuccessRate)
	}
}

func hasSubstring(items []string, sub string) bool {
	for _, s := range items {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
