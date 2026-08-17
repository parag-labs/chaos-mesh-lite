# chaos-mesh-lite: design, trade-offs, and non-goals

Status: accepted
Author: Parag Sawant

Why chaos-mesh-lite is built the way it is. Chaos engineering usually lives in a
one-off drill someone runs by hand and then forgets. The bet here is that the useful
version is a *gate*: inject a fault, assert the SLO still holds, and fail the build if
it doesn't - so resilience is checked every time, not remembered occasionally.

## Problem and goals

You believe your service tolerates some added latency or a fraction of failures - but
"believe" isn't tested until something breaks in production. chaos-mesh-lite turns that
belief into a CI assertion: apply a deterministic fault policy to a stream of observed
call outcomes, compute the resulting SLOs (success rate, p95 latency), and pass or fail
against your objectives. Goals:

1. **Resilience as a pass/fail gate**, runnable in CI, not a manual experiment.
2. **Deterministic faults** - a fixed schedule, not randomness - so a red build is
   reproducible and means the same thing every run.
3. **The same engine in three languages**, because the fault math and SLO computation
   are plain logic that should be identical everywhere.

## Key design decision: deterministic faults, not RNG

This is the choice the whole tool rests on. A chaos experiment driven by a random
number generator gives you a different result each run, which is useless as a gate -
you can't tell a real regression from bad luck. So faults here are a *schedule*:
"add N ms to every call," "fail every Kth call." Given the same input stream and
policy, the outcome is bit-for-bit identical, so a failure is a real signal you can
block a deploy on, and it reproduces on a developer's machine exactly as it did in CI.

The trade-off is that this models *specified* failure modes, not the open-ended
randomness of true production chaos. That's the right trade for a gate: you're
asserting "my SLO survives this defined stress," which is a testable claim, rather than
"my SLO survives whatever the universe throws," which isn't.

## Trade-offs I made on purpose

- **Fault schedule over stochastic injection.** Reproducibility beats realism for a CI
  gate (see above). A stochastic mode with a fixed seed is a reasonable addition for
  exploratory runs, but the default is deterministic on purpose.
- **Operates on an outcome stream, not a live system.** chaos-mesh-lite computes SLOs
  over a sequence of call latencies/outcomes you feed it, rather than injecting faults
  into a running service via a proxy or a sidecar. That keeps it dependency-free and
  trivially portable, and makes the assertion pure and testable. Wiring the same policy
  into a real interceptor is the production step, left as one.
- **p95 as the latency SLO, computed simply.** A nearest-rank percentile over the
  observed latencies - easy to reason about and identical across the three ports. Richer
  percentile estimators (t-digest for huge streams) are a scale concern that doesn't
  change the gate's meaning.

## Why there's no separate benchmark

chaos-mesh-lite *is* the stress tool - it exists to apply stress and assert SLOs, so its
own tests are the "does it hold under fault" evidence. Benchmarking its throughput would
measure a percentile computation over a list, which nobody waits on. The property that
matters is that the SLO math and the fault schedule are correct and reproducible, and
the tri-language test suites pin that down identically.

## Non-goals

- **Not a live fault injector.** No proxy, sidecar, or kernel hooks; it evaluates an
  outcome stream. Real in-process injection is a downstream integration.
- **Not a load generator.** It doesn't produce traffic; you bring the observed calls.
- **Not a full chaos platform.** No experiment scheduling, blast-radius control, or
  automated rollback - it's the assertion primitive that turns a chaos result into a CI
  pass/fail.

Part of [parag-labs](https://github.com/parag-labs) - small, focused tools for building AI systems you can trust.
