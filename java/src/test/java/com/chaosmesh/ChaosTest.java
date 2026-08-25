package com.chaosmesh;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

import org.junit.jupiter.api.Test;

import com.chaosmesh.Chaos.ExperimentResult;
import com.chaosmesh.Chaos.FaultPolicy;
import com.chaosmesh.Chaos.Slo;
import com.chaosmesh.Chaos.SloResult;

class ChaosTest {

    private static List<Double> repeat(double v, int n) {
        List<Double> out = new ArrayList<>(Collections.nCopies(n, v));
        return out;
    }

    @Test
    void percentileNearestRank() {
        List<Double> data = new ArrayList<>();
        for (int i = 1; i <= 100; i++) {
            data.add((double) i);
        }
        assertEquals(95.0, Chaos.percentile(data, 0.95));
        assertEquals(100.0, Chaos.percentile(data, 1.0));
        assertEquals(0.0, Chaos.percentile(new ArrayList<>(), 0.95));
    }

    @Test
    void noFaultAllSucceed() {
        ExperimentResult result = Chaos.runExperiment(repeat(100.0, 50), new FaultPolicy());
        assertEquals(0, result.failures());
        assertEquals(1.0, result.successRate());
        assertEquals(100.0, result.p95LatencyMs());
    }

    @Test
    void latencyInjectionRaisesP95() {
        ExperimentResult result = Chaos.runExperiment(repeat(100.0, 50), new FaultPolicy(250.0, 0));
        assertEquals(350.0, result.p95LatencyMs());
    }

    @Test
    void errorInjectionLowersSuccessRate() {
        ExperimentResult result = Chaos.runExperiment(repeat(100.0, 100), new FaultPolicy(0.0, 10));
        assertEquals(10, result.failures());
        assertEquals(0.9, result.successRate());
    }

    @Test
    void sloPassesWhenHealthy() {
        ExperimentResult result = Chaos.runExperiment(repeat(100.0, 100), new FaultPolicy());
        assertTrue(Chaos.evaluateSlo(result, new Slo(0.99, 200.0)).passed());
    }

    @Test
    void sloFailsOnLatencyBreach() {
        ExperimentResult result = Chaos.runExperiment(repeat(100.0, 100), new FaultPolicy(500.0, 0));
        SloResult outcome = Chaos.evaluateSlo(result, new Slo(0.99, 200.0));
        assertFalse(outcome.passed());
        assertTrue(outcome.breaches().stream().anyMatch(b -> b.contains("p95")));
    }

    @Test
    void sloFailsOnErrorBreach() {
        ExperimentResult result = Chaos.runExperiment(repeat(100.0, 100), new FaultPolicy(0.0, 4));
        SloResult outcome = Chaos.evaluateSlo(result, new Slo(0.99, 1000.0));
        assertFalse(outcome.passed());
        assertTrue(outcome.breaches().stream().anyMatch(b -> b.contains("success_rate")));
    }

    @Test
    void combinedFaults() {
        ExperimentResult result = Chaos.runExperiment(repeat(50.0, 20), new FaultPolicy(100.0, 5));
        assertEquals(4, result.failures());
        assertEquals(150.0, result.p95LatencyMs());
    }
}
