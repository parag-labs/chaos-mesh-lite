import { describe, expect, it } from "vitest";

import { FaultPolicy, Slo, evaluateSlo, percentile, runExperiment } from "./chaos";

function repeat(v: number, n: number): number[] {
  return new Array<number>(n).fill(v);
}

function rangeFloats(lo: number, hi: number): number[] {
  const out: number[] = [];
  for (let i = lo; i <= hi; i++) {
    out.push(i);
  }
  return out;
}

const healthySlo: Slo = { minSuccessRate: 0.99, maxP95LatencyMs: 200.0 };

describe("percentile", () => {
  it("uses nearest rank", () => {
    const data = rangeFloats(1, 100);
    expect(percentile(data, 0.95)).toBe(95.0);
    expect(percentile(data, 1.0)).toBe(100.0);
    expect(percentile([], 0.95)).toBe(0.0);
  });

  it("takes the smallest element at p = 0", () => {
    expect(percentile(rangeFloats(1, 100), 0.0)).toBe(1.0);
  });

  it("handles a single element", () => {
    expect(percentile([42.0], 0.95)).toBe(42.0);
  });

  it("does not mutate its input", () => {
    const data = [3.0, 1.0, 2.0];
    percentile(data, 0.5);
    expect(data).toEqual([3.0, 1.0, 2.0]);
  });
});

describe("runExperiment", () => {
  it("lets every call succeed with no fault policy", () => {
    const result = runExperiment(repeat(100.0, 50), new FaultPolicy());
    expect(result.failures).toBe(0);
    expect(result.successRate).toBe(1.0);
    expect(result.p95LatencyMs).toBe(100.0);
  });

  it("raises p95 when latency is injected", () => {
    const result = runExperiment(repeat(100.0, 50), new FaultPolicy(250.0));
    expect(result.p95LatencyMs).toBe(350.0);
  });

  it("lowers success rate when errors are injected", () => {
    const result = runExperiment(repeat(100.0, 100), new FaultPolicy(0.0, 10));
    expect(result.failures).toBe(10);
    expect(result.successRate).toBe(0.9);
  });

  it("combines latency and error faults", () => {
    const result = runExperiment(repeat(50.0, 20), new FaultPolicy(100.0, 5));
    expect(result.failures).toBe(4);
    expect(result.p95LatencyMs).toBe(150.0);
  });

  it("returns zeros for empty input", () => {
    const result = runExperiment([], new FaultPolicy());
    expect(result.totalCalls).toBe(0);
    expect(result.failures).toBe(0);
    expect(result.successRate).toBe(0.0);
    expect(result.p95LatencyMs).toBe(0.0);
  });

  it("leaves no latency samples when every call fails", () => {
    const result = runExperiment(repeat(100.0, 10), new FaultPolicy(0.0, 1));
    expect(result.failures).toBe(10);
    expect(result.successRate).toBe(0.0);
    expect(result.p95LatencyMs).toBe(0.0);
  });

  it("never fails when failEveryN is zero", () => {
    const result = runExperiment(repeat(10.0, 7), new FaultPolicy(0.0, 0));
    expect(result.failures).toBe(0);
    expect(result.successRate).toBe(1.0);
  });
});

describe("evaluateSlo", () => {
  it("passes and reports no breaches when healthy", () => {
    const result = runExperiment(repeat(100.0, 100), new FaultPolicy());
    const outcome = evaluateSlo(result, healthySlo);
    expect(outcome.passed).toBe(true);
    expect(outcome.breaches).toHaveLength(0);
  });

  it("fails on a latency breach", () => {
    const result = runExperiment(repeat(100.0, 100), new FaultPolicy(500.0));
    const outcome = evaluateSlo(result, healthySlo);
    expect(outcome.passed).toBe(false);
    expect(outcome.breaches.some((b) => b.includes("p95"))).toBe(true);
  });

  it("fails on an error-rate breach", () => {
    const result = runExperiment(repeat(100.0, 100), new FaultPolicy(0.0, 4));
    const outcome = evaluateSlo(result, { minSuccessRate: 0.99, maxP95LatencyMs: 1000.0 });
    expect(outcome.passed).toBe(false);
    expect(outcome.breaches.some((b) => b.includes("success_rate"))).toBe(true);
  });

  it("reports the success-rate breach before the latency breach", () => {
    const result = runExperiment(repeat(100.0, 100), new FaultPolicy(500.0, 4));
    const outcome = evaluateSlo(result, healthySlo);
    expect(outcome.passed).toBe(false);
    expect(outcome.breaches).toHaveLength(2);
    expect(outcome.breaches[0]).toContain("success_rate");
    expect(outcome.breaches[1]).toContain("p95");
  });

  it("formats the breach message like the reference", () => {
    const result = runExperiment(repeat(100.0, 100), new FaultPolicy(500.0));
    const outcome = evaluateSlo(result, healthySlo);
    expect(outcome.breaches[0]).toBe("p95 600.00ms > 200.00ms");
  });
});
