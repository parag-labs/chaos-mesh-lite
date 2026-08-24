"""Command line front end for ChaosMeshLite.

Point it at a file of observed latencies (one per line), tell it what faults to
inject and what SLO to hold, and it runs the experiment. Exits 1 if the SLO
breaks under the injected faults -- so a flaky-under-stress service fails the build.

    python -m src.cli latencies.txt --added-latency 300 --fail-every 20 \
        --min-success 0.99 --max-p95 250
"""

from __future__ import annotations

import argparse
import sys

from chaos import FaultPolicy, Slo, evaluate_slo, run_experiment


def read_numbers(path: str) -> list[float]:
    values = []
    with open(path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                values.append(float(line))
    return values


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="chaosmesh")
    parser.add_argument("latencies", help="observed latencies (ms), one per line")
    parser.add_argument("--added-latency", type=float, default=0.0)
    parser.add_argument("--fail-every", type=int, default=0)
    parser.add_argument("--min-success", type=float, default=0.99)
    parser.add_argument("--max-p95", type=float, default=250.0)
    args = parser.parse_args(argv)

    policy = FaultPolicy(added_latency_ms=args.added_latency, fail_every_n=args.fail_every)
    result = run_experiment(read_numbers(args.latencies), policy)
    outcome = evaluate_slo(result, Slo(args.min_success, args.max_p95))

    print(f"calls        {result.total_calls}")
    print(f"failures     {result.failures}")
    print(f"success_rate {result.success_rate}")
    print(f"p95_latency  {result.p95_latency_ms}ms")
    if outcome.passed:
        print("SLO held under chaos")
        return 0
    print("SLO BREACHED:")
    for b in outcome.breaches:
        print(f"  - {b}")
    return 1


if __name__ == "__main__":
    sys.exit(main())
