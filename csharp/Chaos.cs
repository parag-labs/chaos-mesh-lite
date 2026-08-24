namespace ChaosMeshLite;

public readonly record struct FaultPolicy(double AddedLatencyMs = 0.0, int FailEveryN = 0);

public readonly record struct ExperimentResult(
    int TotalCalls, int Failures, double SuccessRate, double P95LatencyMs);

public readonly record struct Slo(double MinSuccessRate, double MaxP95LatencyMs);

public sealed record SloResult(bool Passed, double SuccessRate, double P95LatencyMs, List<string> Breaches);

/// <summary>Chaos experiments as CI-runnable SLO assertions (deterministic fault schedule).</summary>
public static class Chaos
{
    public static double Percentile(IReadOnlyList<double> values, double p)
    {
        if (values.Count == 0)
        {
            return 0.0;
        }

        var ordered = values.OrderBy(v => v).ToList();
        var rank = (int)Math.Ceiling(p * ordered.Count);
        var idx = Math.Min(Math.Max(rank - 1, 0), ordered.Count - 1);
        return ordered[idx];
    }

    public static ExperimentResult RunExperiment(IReadOnlyList<double> baseLatenciesMs, FaultPolicy policy)
    {
        var latencies = new List<double>();
        var failures = 0;
        for (var i = 0; i < baseLatenciesMs.Count; i++)
        {
            var oneIndexed = i + 1;
            var isFailure = policy.FailEveryN > 0 && oneIndexed % policy.FailEveryN == 0;
            if (isFailure)
            {
                failures++;
                continue;
            }

            latencies.Add(baseLatenciesMs[i] + policy.AddedLatencyMs);
        }

        var total = baseLatenciesMs.Count;
        var successes = total - failures;
        return new ExperimentResult(
            total,
            failures,
            total == 0 ? 0.0 : Math.Round((double)successes / total, 6),
            Math.Round(Percentile(latencies, 0.95), 6));
    }

    public static SloResult EvaluateSlo(ExperimentResult result, Slo slo)
    {
        var breaches = new List<string>();
        if (result.SuccessRate < slo.MinSuccessRate)
        {
            breaches.Add($"success_rate {result.SuccessRate:F4} < {slo.MinSuccessRate:F4}");
        }

        if (result.P95LatencyMs > slo.MaxP95LatencyMs)
        {
            breaches.Add($"p95 {result.P95LatencyMs:F2}ms > {slo.MaxP95LatencyMs:F2}ms");
        }

        return new SloResult(breaches.Count == 0, result.SuccessRate, result.P95LatencyMs, breaches);
    }
}
