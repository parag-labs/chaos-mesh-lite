//! Deterministic fault injection and SLO evaluation.

/// Faults injected into an experiment. The [`Default`] value injects nothing.
#[derive(Debug, Clone, Copy, PartialEq, Default)]
pub struct FaultPolicy {
    /// Latency added to every non-failed call.
    pub added_latency_ms: f64,
    /// Fails calls whose 1-indexed position is a multiple of this value.
    /// A value of 0 means no calls ever fail.
    pub fail_every_n: usize,
}

/// The measured outcome of a single experiment run.
#[derive(Debug, Clone, Copy, PartialEq)]
pub struct ExperimentResult {
    /// Number of observed calls, including failures.
    pub total_calls: usize,
    /// Number of injected failures.
    pub failures: usize,
    /// successes/total rounded to six decimals (0 when there were no calls).
    pub success_rate: f64,
    /// 95th-percentile latency of the successful calls.
    pub p95_latency_ms: f64,
}

/// A Service Level Objective to assert against an [`ExperimentResult`].
#[derive(Debug, Clone, Copy, PartialEq)]
pub struct Slo {
    /// Lowest acceptable success rate.
    pub min_success_rate: f64,
    /// Highest acceptable p95 latency.
    pub max_p95_latency_ms: f64,
}

/// Whether an experiment met its SLO and, if not, why.
#[derive(Debug, Clone, PartialEq)]
pub struct SloResult {
    /// True when there are no breaches.
    pub passed: bool,
    /// Success rate of the evaluated result.
    pub success_rate: f64,
    /// p95 latency of the evaluated result.
    pub p95_latency_ms: f64,
    /// Human-readable breach reasons (empty when passed).
    pub breaches: Vec<String>,
}

/// Returns the nearest-rank percentile of `values` for `p` in `[0, 1]`.
///
/// An empty slice yields `0.0`. The input slice is not modified.
#[must_use]
pub fn percentile(values: &[f64], p: f64) -> f64 {
    if values.is_empty() {
        return 0.0;
    }
    let mut ordered = values.to_vec();
    ordered.sort_by(f64::total_cmp);
    let rank = (p * ordered.len() as f64).ceil() as isize;
    let idx = (rank - 1).clamp(0, ordered.len() as isize - 1) as usize;
    ordered[idx]
}

/// Applies `policy` to `base_latencies_ms` and measures the outcome.
///
/// Failed calls contribute no latency sample; every surviving call has
/// `added_latency_ms` added to its base latency.
#[must_use]
pub fn run_experiment(base_latencies_ms: &[f64], policy: FaultPolicy) -> ExperimentResult {
    let mut latencies: Vec<f64> = Vec::new();
    let mut failures = 0usize;
    for (i, &base) in base_latencies_ms.iter().enumerate() {
        let one_indexed = i + 1;
        let is_failure = policy.fail_every_n > 0 && one_indexed % policy.fail_every_n == 0;
        if is_failure {
            failures += 1;
            continue;
        }
        latencies.push(base + policy.added_latency_ms);
    }

    let total = base_latencies_ms.len();
    let successes = total - failures;
    let success_rate = if total == 0 {
        0.0
    } else {
        round6(successes as f64 / total as f64)
    };
    ExperimentResult {
        total_calls: total,
        failures,
        success_rate,
        p95_latency_ms: round6(percentile(&latencies, 0.95)),
    }
}

/// Checks `result` against `slo` and returns a pass/fail verdict with breach
/// reasons. The success-rate breach (if any) is reported before the latency
/// breach.
#[must_use]
pub fn evaluate_slo(result: ExperimentResult, slo: Slo) -> SloResult {
    let mut breaches: Vec<String> = Vec::new();
    if result.success_rate < slo.min_success_rate {
        breaches.push(format!(
            "success_rate {:.4} < {:.4}",
            result.success_rate, slo.min_success_rate
        ));
    }
    if result.p95_latency_ms > slo.max_p95_latency_ms {
        breaches.push(format!(
            "p95 {:.2}ms > {:.2}ms",
            result.p95_latency_ms, slo.max_p95_latency_ms
        ));
    }
    SloResult {
        passed: breaches.is_empty(),
        success_rate: result.success_rate,
        p95_latency_ms: result.p95_latency_ms,
        breaches,
    }
}

fn round6(v: f64) -> f64 {
    (v * 1_000_000.0).round() / 1_000_000.0
}
