/**
 * ChaosMeshLite: chaos experiments as CI-runnable SLO assertions.
 *
 * Inject deterministic faults (added latency, periodic errors) into a stream of
 * observed call latencies, then assert that Service Level Objectives still hold.
 * The fault schedule is deterministic (not RNG) so results are reproducible and
 * identical across every language port.
 */

/** Faults injected into an experiment. The defaults inject nothing. */
export class FaultPolicy {
  /** Latency added to every non-failed call. */
  readonly addedLatencyMs: number;
  /**
   * Fails calls whose 1-indexed position is a multiple of this value.
   * A value of 0 means no calls ever fail.
   */
  readonly failEveryN: number;

  constructor(addedLatencyMs = 0.0, failEveryN = 0) {
    this.addedLatencyMs = addedLatencyMs;
    this.failEveryN = failEveryN;
  }
}

/** The measured outcome of a single experiment run. */
export interface ExperimentResult {
  /** Number of observed calls, including failures. */
  readonly totalCalls: number;
  /** Number of injected failures. */
  readonly failures: number;
  /** successes/total rounded to six decimals (0 when there were no calls). */
  readonly successRate: number;
  /** 95th-percentile latency of the successful calls. */
  readonly p95LatencyMs: number;
}

/** A Service Level Objective to assert against an {@link ExperimentResult}. */
export interface Slo {
  /** Lowest acceptable success rate. */
  readonly minSuccessRate: number;
  /** Highest acceptable p95 latency. */
  readonly maxP95LatencyMs: number;
}

/** Whether an experiment met its SLO and, if not, why. */
export interface SloResult {
  /** True when there are no breaches. */
  readonly passed: boolean;
  /** Success rate of the evaluated result. */
  readonly successRate: number;
  /** p95 latency of the evaluated result. */
  readonly p95LatencyMs: number;
  /** Human-readable breach reasons (empty when passed). */
  readonly breaches: string[];
}

/**
 * Returns the nearest-rank percentile of `values` for `p` in [0, 1].
 * An empty array yields 0. The input array is not modified.
 */
export function percentile(values: number[], p: number): number {
  if (values.length === 0) {
    return 0.0;
  }
  const ordered = [...values].sort((a, b) => a - b);
  const rank = Math.ceil(p * ordered.length);
  const idx = Math.min(Math.max(rank - 1, 0), ordered.length - 1);
  return ordered[idx];
}

/**
 * Applies `policy` to `baseLatenciesMs` and measures the outcome. Failed calls
 * contribute no latency sample; every surviving call has `addedLatencyMs` added
 * to its base latency.
 */
export function runExperiment(baseLatenciesMs: number[], policy: FaultPolicy): ExperimentResult {
  const latencies: number[] = [];
  let failures = 0;
  for (let i = 0; i < baseLatenciesMs.length; i++) {
    const oneIndexed = i + 1;
    const isFailure = policy.failEveryN > 0 && oneIndexed % policy.failEveryN === 0;
    if (isFailure) {
      failures++;
      continue;
    }
    latencies.push(baseLatenciesMs[i] + policy.addedLatencyMs);
  }

  const total = baseLatenciesMs.length;
  const successes = total - failures;
  const successRate = total === 0 ? 0.0 : round6(successes / total);
  return {
    totalCalls: total,
    failures,
    successRate,
    p95LatencyMs: round6(percentile(latencies, 0.95)),
  };
}

/**
 * Checks `result` against `slo` and returns a pass/fail verdict with breach
 * reasons. The success-rate breach (if any) is reported before the latency
 * breach.
 */
export function evaluateSlo(result: ExperimentResult, slo: Slo): SloResult {
  const breaches: string[] = [];
  if (result.successRate < slo.minSuccessRate) {
    breaches.push(
      `success_rate ${result.successRate.toFixed(4)} < ${slo.minSuccessRate.toFixed(4)}`,
    );
  }
  if (result.p95LatencyMs > slo.maxP95LatencyMs) {
    breaches.push(
      `p95 ${result.p95LatencyMs.toFixed(2)}ms > ${slo.maxP95LatencyMs.toFixed(2)}ms`,
    );
  }
  return {
    passed: breaches.length === 0,
    successRate: result.successRate,
    p95LatencyMs: result.p95LatencyMs,
    breaches,
  };
}

function round6(v: number): number {
  return Math.round(v * 1_000_000.0) / 1_000_000.0;
}
