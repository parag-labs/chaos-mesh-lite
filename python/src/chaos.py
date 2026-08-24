"""Chaos experiments as CI-runnable SLO assertions.

Inject deterministic faults (added latency, periodic errors) into a stream of
observed call latencies, then assert that Service Level Objectives still hold. If
they don't, the experiment fails -- resilience testing as a pass/fail gate, not a
one-off manual drill.

Deterministic by design (fault schedule, not RNG) so results are reproducible and
identical across Python, C#, and Java.
"""

from __future__ import annotations

import math
from dataclasses import dataclass


@dataclass(frozen=True)
class FaultPolicy:
    added_latency_ms: float = 0.0
    fail_every_n: int = 0  # 0 = never; else fail calls where (index % n == 0), 1-indexed


@dataclass(frozen=True)
class ExperimentResult:
    total_calls: int
    failures: int
    success_rate: float
    p95_latency_ms: float


@dataclass(frozen=True)
class Slo:
    min_success_rate: float
    max_p95_latency_ms: float


@dataclass(frozen=True)
class SloResult:
    passed: bool
    success_rate: float
    p95_latency_ms: float
    breaches: list[str]


def percentile(values: list[float], p: float) -> float:
    """Nearest-rank percentile (p in [0,1])."""
    if not values:
        return 0.0
    ordered = sorted(values)
    rank = math.ceil(p * len(ordered))
    idx = min(max(rank - 1, 0), len(ordered) - 1)
    return ordered[idx]


def run_experiment(base_latencies_ms: list[float], policy: FaultPolicy) -> ExperimentResult:
    latencies: list[float] = []
    failures = 0
    for i, base in enumerate(base_latencies_ms, start=1):
        is_failure = policy.fail_every_n > 0 and i % policy.fail_every_n == 0
        if is_failure:
            failures += 1
            continue  # failed calls don't contribute a latency sample
        latencies.append(base + policy.added_latency_ms)

    total = len(base_latencies_ms)
    successes = total - failures
    return ExperimentResult(
        total_calls=total,
        failures=failures,
        success_rate=round(successes / total, 6) if total else 0.0,
        p95_latency_ms=round(percentile(latencies, 0.95), 6),
    )


def evaluate_slo(result: ExperimentResult, slo: Slo) -> SloResult:
    breaches = []
    if result.success_rate < slo.min_success_rate:
        breaches.append(
            f"success_rate {result.success_rate:.4f} < {slo.min_success_rate:.4f}"
        )
    if result.p95_latency_ms > slo.max_p95_latency_ms:
        breaches.append(
            f"p95 {result.p95_latency_ms:.2f}ms > {slo.max_p95_latency_ms:.2f}ms"
        )
    return SloResult(
        passed=not breaches,
        success_rate=result.success_rate,
        p95_latency_ms=result.p95_latency_ms,
        breaches=breaches,
    )
