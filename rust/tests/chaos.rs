use chaos_mesh_lite::{evaluate_slo, percentile, run_experiment, FaultPolicy, Slo};

fn repeat(v: f64, n: usize) -> Vec<f64> {
    vec![v; n]
}

fn range_floats(lo: i64, hi: i64) -> Vec<f64> {
    (lo..=hi).map(|i| i as f64).collect()
}

#[test]
fn percentile_nearest_rank() {
    let data = range_floats(1, 100);
    assert_eq!(percentile(&data, 0.95), 95.0);
    assert_eq!(percentile(&data, 1.0), 100.0);
    assert_eq!(percentile(&[], 0.95), 0.0);
}

#[test]
fn percentile_zero_takes_smallest() {
    assert_eq!(percentile(&range_floats(1, 100), 0.0), 1.0);
}

#[test]
fn percentile_single_element() {
    assert_eq!(percentile(&[42.0], 0.95), 42.0);
}

#[test]
fn percentile_does_not_mutate_input() {
    let data = vec![3.0, 1.0, 2.0];
    let _ = percentile(&data, 0.5);
    assert_eq!(data, vec![3.0, 1.0, 2.0]);
}

#[test]
fn no_fault_all_succeed() {
    let result = run_experiment(&repeat(100.0, 50), FaultPolicy::default());
    assert_eq!(result.failures, 0);
    assert_eq!(result.success_rate, 1.0);
    assert_eq!(result.p95_latency_ms, 100.0);
}

#[test]
fn latency_injection_raises_p95() {
    let policy = FaultPolicy {
        added_latency_ms: 250.0,
        ..FaultPolicy::default()
    };
    let result = run_experiment(&repeat(100.0, 50), policy);
    assert_eq!(result.p95_latency_ms, 350.0);
}

#[test]
fn error_injection_lowers_success_rate() {
    let policy = FaultPolicy {
        fail_every_n: 10,
        ..FaultPolicy::default()
    };
    let result = run_experiment(&repeat(100.0, 100), policy);
    assert_eq!(result.failures, 10);
    assert_eq!(result.success_rate, 0.9);
}

#[test]
fn slo_passes_when_healthy() {
    let result = run_experiment(&repeat(100.0, 100), FaultPolicy::default());
    let outcome = evaluate_slo(
        result,
        Slo {
            min_success_rate: 0.99,
            max_p95_latency_ms: 200.0,
        },
    );
    assert!(outcome.passed);
    assert!(outcome.breaches.is_empty());
}

#[test]
fn slo_fails_on_latency_breach() {
    let policy = FaultPolicy {
        added_latency_ms: 500.0,
        ..FaultPolicy::default()
    };
    let result = run_experiment(&repeat(100.0, 100), policy);
    let outcome = evaluate_slo(
        result,
        Slo {
            min_success_rate: 0.99,
            max_p95_latency_ms: 200.0,
        },
    );
    assert!(!outcome.passed);
    assert!(outcome.breaches.iter().any(|b| b.contains("p95")));
}

#[test]
fn slo_fails_on_error_breach() {
    let policy = FaultPolicy {
        fail_every_n: 4,
        ..FaultPolicy::default()
    };
    let result = run_experiment(&repeat(100.0, 100), policy);
    let outcome = evaluate_slo(
        result,
        Slo {
            min_success_rate: 0.99,
            max_p95_latency_ms: 1000.0,
        },
    );
    assert!(!outcome.passed);
    assert!(outcome.breaches.iter().any(|b| b.contains("success_rate")));
}

#[test]
fn combined_faults() {
    let policy = FaultPolicy {
        added_latency_ms: 100.0,
        fail_every_n: 5,
    };
    let result = run_experiment(&repeat(50.0, 20), policy);
    assert_eq!(result.failures, 4);
    assert_eq!(result.p95_latency_ms, 150.0);
}

#[test]
fn empty_base_latencies() {
    let result = run_experiment(&[], FaultPolicy::default());
    assert_eq!(result.total_calls, 0);
    assert_eq!(result.failures, 0);
    assert_eq!(result.success_rate, 0.0);
    assert_eq!(result.p95_latency_ms, 0.0);
}

#[test]
fn all_calls_fail_leave_no_latencies() {
    let policy = FaultPolicy {
        fail_every_n: 1,
        ..FaultPolicy::default()
    };
    let result = run_experiment(&repeat(100.0, 10), policy);
    assert_eq!(result.failures, 10);
    assert_eq!(result.success_rate, 0.0);
    assert_eq!(result.p95_latency_ms, 0.0);
}

#[test]
fn both_breaches_reported_success_rate_first() {
    let policy = FaultPolicy {
        added_latency_ms: 500.0,
        fail_every_n: 4,
    };
    let result = run_experiment(&repeat(100.0, 100), policy);
    let outcome = evaluate_slo(
        result,
        Slo {
            min_success_rate: 0.99,
            max_p95_latency_ms: 200.0,
        },
    );
    assert!(!outcome.passed);
    assert_eq!(outcome.breaches.len(), 2);
    assert!(outcome.breaches[0].contains("success_rate"));
    assert!(outcome.breaches[1].contains("p95"));
}

#[test]
fn breach_message_format() {
    let policy = FaultPolicy {
        added_latency_ms: 500.0,
        ..FaultPolicy::default()
    };
    let result = run_experiment(&repeat(100.0, 100), policy);
    let outcome = evaluate_slo(
        result,
        Slo {
            min_success_rate: 0.99,
            max_p95_latency_ms: 200.0,
        },
    );
    assert_eq!(outcome.breaches[0], "p95 600.00ms > 200.00ms");
}

#[test]
fn fail_every_n_zero_never_fails() {
    let policy = FaultPolicy {
        fail_every_n: 0,
        ..FaultPolicy::default()
    };
    let result = run_experiment(&repeat(10.0, 7), policy);
    assert_eq!(result.failures, 0);
    assert_eq!(result.success_rate, 1.0);
}
