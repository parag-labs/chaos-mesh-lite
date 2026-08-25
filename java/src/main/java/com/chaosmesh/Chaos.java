package com.chaosmesh;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

/** Chaos experiments as CI-runnable SLO assertions (deterministic fault schedule). */
public final class Chaos {

    public record FaultPolicy(double addedLatencyMs, int failEveryN) {
        public FaultPolicy() {
            this(0.0, 0);
        }
    }

    public record ExperimentResult(int totalCalls, int failures, double successRate, double p95LatencyMs) {
    }

    public record Slo(double minSuccessRate, double maxP95LatencyMs) {
    }

    public record SloResult(boolean passed, double successRate, double p95LatencyMs, List<String> breaches) {
    }

    public static double percentile(List<Double> values, double p) {
        if (values.isEmpty()) {
            return 0.0;
        }
        List<Double> ordered = new ArrayList<>(values);
        Collections.sort(ordered);
        int rank = (int) Math.ceil(p * ordered.size());
        int idx = Math.min(Math.max(rank - 1, 0), ordered.size() - 1);
        return ordered.get(idx);
    }

    public static ExperimentResult runExperiment(List<Double> baseLatenciesMs, FaultPolicy policy) {
        List<Double> latencies = new ArrayList<>();
        int failures = 0;
        for (int i = 0; i < baseLatenciesMs.size(); i++) {
            int oneIndexed = i + 1;
            boolean isFailure = policy.failEveryN() > 0 && oneIndexed % policy.failEveryN() == 0;
            if (isFailure) {
                failures++;
                continue;
            }
            latencies.add(baseLatenciesMs.get(i) + policy.addedLatencyMs());
        }
        int total = baseLatenciesMs.size();
        int successes = total - failures;
        double successRate = total == 0 ? 0.0 : round6((double) successes / total);
        return new ExperimentResult(total, failures, successRate, round6(percentile(latencies, 0.95)));
    }

    public static SloResult evaluateSlo(ExperimentResult result, Slo slo) {
        List<String> breaches = new ArrayList<>();
        if (result.successRate() < slo.minSuccessRate()) {
            breaches.add(String.format("success_rate %.4f < %.4f", result.successRate(), slo.minSuccessRate()));
        }
        if (result.p95LatencyMs() > slo.maxP95LatencyMs()) {
            breaches.add(String.format("p95 %.2fms > %.2fms", result.p95LatencyMs(), slo.maxP95LatencyMs()));
        }
        return new SloResult(breaches.isEmpty(), result.successRate(), result.p95LatencyMs(), breaches);
    }

    static double round6(double v) {
        return Math.round(v * 1_000_000.0) / 1_000_000.0;
    }

    private Chaos() {
    }
}
