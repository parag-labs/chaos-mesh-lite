using Xunit;

namespace ChaosMeshLite.Tests;

public class ChaosTests
{
    private static List<double> Repeat(double v, int n) => Enumerable.Repeat(v, n).ToList();

    [Fact]
    public void PercentileNearestRank()
    {
        var data = Enumerable.Range(1, 100).Select(i => (double)i).ToList();
        Assert.Equal(95.0, Chaos.Percentile(data, 0.95));
        Assert.Equal(100.0, Chaos.Percentile(data, 1.0));
        Assert.Equal(0.0, Chaos.Percentile([], 0.95));
    }

    [Fact]
    public void NoFaultAllSucceed()
    {
        var result = Chaos.RunExperiment(Repeat(100.0, 50), new FaultPolicy());
        Assert.Equal(0, result.Failures);
        Assert.Equal(1.0, result.SuccessRate);
        Assert.Equal(100.0, result.P95LatencyMs);
    }

    [Fact]
    public void LatencyInjectionRaisesP95()
    {
        var result = Chaos.RunExperiment(Repeat(100.0, 50), new FaultPolicy(AddedLatencyMs: 250.0));
        Assert.Equal(350.0, result.P95LatencyMs);
    }

    [Fact]
    public void ErrorInjectionLowersSuccessRate()
    {
        var result = Chaos.RunExperiment(Repeat(100.0, 100), new FaultPolicy(FailEveryN: 10));
        Assert.Equal(10, result.Failures);
        Assert.Equal(0.9, result.SuccessRate);
    }

    [Fact]
    public void SloPassesWhenHealthy()
    {
        var result = Chaos.RunExperiment(Repeat(100.0, 100), new FaultPolicy());
        var slo = new Slo(0.99, 200.0);
        Assert.True(Chaos.EvaluateSlo(result, slo).Passed);
    }

    [Fact]
    public void SloFailsOnLatencyBreach()
    {
        var result = Chaos.RunExperiment(Repeat(100.0, 100), new FaultPolicy(AddedLatencyMs: 500.0));
        var outcome = Chaos.EvaluateSlo(result, new Slo(0.99, 200.0));
        Assert.False(outcome.Passed);
        Assert.Contains(outcome.Breaches, b => b.Contains("p95"));
    }

    [Fact]
    public void SloFailsOnErrorBreach()
    {
        var result = Chaos.RunExperiment(Repeat(100.0, 100), new FaultPolicy(FailEveryN: 4));
        var outcome = Chaos.EvaluateSlo(result, new Slo(0.99, 1000.0));
        Assert.False(outcome.Passed);
        Assert.Contains(outcome.Breaches, b => b.Contains("success_rate"));
    }

    [Fact]
    public void CombinedFaults()
    {
        var result = Chaos.RunExperiment(Repeat(50.0, 20), new FaultPolicy(AddedLatencyMs: 100.0, FailEveryN: 5));
        Assert.Equal(4, result.Failures);
        Assert.Equal(150.0, result.P95LatencyMs);
    }
}
