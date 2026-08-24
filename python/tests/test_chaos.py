"""ChaosMeshLite tests: fault injection changes outcomes; SLO gate catches breaches."""

from chaos import FaultPolicy, Slo, evaluate_slo, percentile, run_experiment


def test_percentile_nearest_rank():
    data = [float(i) for i in range(1, 101)]  # 1..100
    assert percentile(data, 0.95) == 95.0
    assert percentile(data, 1.0) == 100.0
    assert percentile([], 0.95) == 0.0


def test_no_fault_all_succeed():
    base = [100.0] * 50
    result = run_experiment(base, FaultPolicy())
    assert result.failures == 0
    assert result.success_rate == 1.0
    assert result.p95_latency_ms == 100.0


def test_latency_injection_raises_p95():
    base = [100.0] * 50
    result = run_experiment(base, FaultPolicy(added_latency_ms=250.0))
    assert result.p95_latency_ms == 350.0


def test_error_injection_lowers_success_rate():
    base = [100.0] * 100
    # fail every 10th call -> 10 failures -> 90% success.
    result = run_experiment(base, FaultPolicy(fail_every_n=10))
    assert result.failures == 10
    assert result.success_rate == 0.9


def test_slo_passes_when_healthy():
    result = run_experiment([100.0] * 100, FaultPolicy())
    slo = Slo(min_success_rate=0.99, max_p95_latency_ms=200.0)
    assert evaluate_slo(result, slo).passed is True


def test_slo_fails_on_latency_breach():
    result = run_experiment([100.0] * 100, FaultPolicy(added_latency_ms=500.0))
    slo = Slo(min_success_rate=0.99, max_p95_latency_ms=200.0)
    outcome = evaluate_slo(result, slo)
    assert outcome.passed is False
    assert any("p95" in b for b in outcome.breaches)


def test_slo_fails_on_error_breach():
    result = run_experiment([100.0] * 100, FaultPolicy(fail_every_n=4))  # 25% failures
    slo = Slo(min_success_rate=0.99, max_p95_latency_ms=1000.0)
    outcome = evaluate_slo(result, slo)
    assert outcome.passed is False
    assert any("success_rate" in b for b in outcome.breaches)


def test_combined_faults():
    result = run_experiment([50.0] * 20, FaultPolicy(added_latency_ms=100.0, fail_every_n=5))
    assert result.failures == 4          # calls 5,10,15,20
    assert result.p95_latency_ms == 150.0
