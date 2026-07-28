#!/usr/bin/env python3
"""Convert base/base snapshot benchmark results into report input files."""

import argparse
import json
from datetime import datetime, timezone
from pathlib import Path


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Import base-bench snapshot result JSON files for the visualizer"
    )
    parser.add_argument("results", nargs="+", type=Path, help="base-bench result JSON")
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=Path("output"),
        help="visualizer output directory (default: output)",
    )
    parser.add_argument(
        "--benchmark-run",
        default="snapshot-throughput",
        help="grouping ID used by the report (default: snapshot-throughput)",
    )
    parser.add_argument(
        "--client-version",
        default="local",
        help="client build identifier stored in metadata (default: local)",
    )
    return parser.parse_args()


def metric_samples(blocks: list[dict], interval_ms: int) -> list[dict]:
    block_seconds = interval_ms / 1_000
    return [
        {
            "BlockNumber": block["number"],
            "ExecutionMetrics": {
                "gas/per_block": block["gas_used"],
                "gas/per_second": block["gas_used"] / block_seconds,
                "transactions/per_block": block["transaction_count"],
                "transactions/per_second": block["transaction_count"] / block_seconds,
            },
        }
        for block in blocks
    ]


def canonical_rate(blocks: list[dict], interval_ms: int, key: str) -> float:
    if not blocks:
        return 0
    duration_seconds = len(blocks) * interval_ms / 1_000
    return sum(block[key] for block in blocks) / duration_seconds


def workload_name(result: dict) -> str:
    ratio = result["load_test"]["config"].get("fresh_recipient_ratio", 0)
    return "fresh-account" if ratio == 1 else "existing-account"


def import_result(
    source: Path, output_root: Path, benchmark_run: str, client_version: str
) -> dict:
    result = json.loads(source.read_text())
    interval_ms = result["block_interval_ms"]
    workload = workload_name(result)
    run_name = source.stem
    run_dir = output_root / run_name
    run_dir.mkdir(parents=True, exist_ok=True)

    blocks = result["blocks"]
    validator_blocks = result["validator_blocks"]
    (run_dir / "metrics-sequencer.json").write_text(
        json.dumps(metric_samples(blocks, interval_ms), indent=2) + "\n"
    )
    (run_dir / "metrics-validator.json").write_text(
        json.dumps(metric_samples(validator_blocks, interval_ms), indent=2) + "\n"
    )

    load_test = result["load_test"]
    load_test["block_range"] = {
        "first_block": blocks[0]["number"],
        "last_block": blocks[-1]["number"],
        "block_count": len(blocks),
    }
    (run_dir / "load-test-result.json").write_text(
        json.dumps(load_test, indent=2) + "\n"
    )

    load_throughput = load_test["throughput"]
    created_at = datetime.fromtimestamp(source.stat().st_mtime, timezone.utc).isoformat()
    return {
        "id": run_name,
        "sourceFile": source.name,
        "testName": "Base mainnet snapshot throughput",
        "testDescription": "Saturated block production from an existing Base mainnet datadir",
        "outputDir": run_name,
        "createdAt": created_at,
        "testConfig": {
            "BenchmarkRun": benchmark_run,
            "BlockTimeMilliseconds": interval_ms,
            "GasLimit": blocks[0]["gas_limit"],
            "Workload": workload,
            "ClientVersion": client_version,
        },
        "result": {
            "success": load_test.get("error") is None,
            "complete": True,
            "clientVersion": client_version,
            "sequencerMetrics": {
                "gasPerSecond": canonical_rate(blocks, interval_ms, "gas_used"),
                "transactionsPerSecond": canonical_rate(
                    blocks, interval_ms, "transaction_count"
                ),
            },
            "validatorMetrics": {
                "gasPerSecond": canonical_rate(
                    validator_blocks, interval_ms, "gas_used"
                ),
                "transactionsPerSecond": canonical_rate(
                    validator_blocks, interval_ms, "transaction_count"
                ),
            },
            "loadTestMetrics": {
                "gasPerSecond": load_throughput["gps"],
                "transactionsPerSecond": load_throughput["tps"],
                "submitted": load_throughput["total_submitted"],
                "confirmed": load_throughput["total_confirmed"],
                "failed": load_throughput["total_failed"],
                "reverted": load_throughput.get("total_reverted", 0),
            },
            "artifacts": {"loadTestResult": "load-test-result.json"},
        },
    }


def main() -> None:
    args = parse_args()
    args.output_dir.mkdir(parents=True, exist_ok=True)
    runs = [
        import_result(
            source, args.output_dir, args.benchmark_run, args.client_version
        )
        for source in args.results
    ]
    (args.output_dir / "metadata.json").write_text(
        json.dumps({"runs": runs}, indent=2) + "\n"
    )
    print(
        f"Imported {len(runs)} run(s). Open /run-comparison/{args.benchmark_run} in the report."
    )


if __name__ == "__main__":
    main()
