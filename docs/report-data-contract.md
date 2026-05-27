# Report data contract

This document describes what the `report/` web UI in this repo expects
from whatever produces benchmark data. The intent is to let `base/benchmark`
become a **pure visualization tool** — any runner (Go, Rust, future
monorepo) can produce data the report consumes, as long as it follows
this contract.

The current Go runner in `runner/` is one producer; the future Rust
runner in the Coinbase monorepo will be another. Both write the same
shape into the same S3 layout.

## TL;DR

- One S3 bucket. Under it: `metadata/` directory of small JSON files
  describing runs, and `<outputDir>/` directories holding per-block
  metrics for each run.
- The report-api (`protocols/base-benchmarking/report-api`) merges all
  `metadata/*.json` files into a single `GET /output/metadata.json`
  response. The UI consumes that one response.
- To add a benchmark run: drop one JSON file into `metadata/` and the
  per-block metrics under its `outputDir`. No central file to update.
- To compare across versions, time windows, or any other axis: write
  the axis value into `testConfig` and the report-api groups
  automatically.

## S3 layout

```
s3://<bucket>/
└── <outputDir>/                         # one directory per inner run
    ├── metadata.json                    # this one run's metadata
    ├── metrics-sequencer.json           # per-block sequencer metrics
    ├── metrics-validator.json           # per-block validator metrics
    └── metrics-<other-role>.json
```

Each run owns one prefix. The presence of `metadata.json` under a
prefix is the **commit signal**: the report-api treats a prefix
without one as an in-progress (or aborted) run and ignores it. The
producer must therefore upload metrics files first and `metadata.json`
last.

> **Legacy note**: prior to the per-run-directory cutover, the layout
> was a central `metadata/metadata-<timestamp>.json` directory plus
> `<outputDir>/` subdirectories. The report-api ignores anything
> under the top-level `metadata/` prefix. The `backend migrate`
> command splits any remaining legacy files into the new layout —
> see the cutover runbook in the project NOTES.

### `<outputDir>/metadata.json`

Every file is a **small JSON document** with this shape:

```json
{
  "runs": [
    {
      "id": "test-1771965834150009",
      "sourceFile": "./mainnet-config.yml",
      "outputDir": "test-1771965834150009-1",
      "testName": "Base Mainnet Performance Benchmark",
      "testDescription": "...",
      "testConfig": {
        "BenchmarkRun": "test-1771965834150009",
        "BlockTimeMilliseconds": 1000,
        "GasLimit": 200000000,
        "NodeType": "builder",
        "TransactionPayload": "transfer-only",
        "ClientVersion": "base-reth-node/v1.11.3-2ac58a2"
      },
      "result": {
        "success": true,
        "complete": true,
        "clientVersion": "base-reth-node/v1.11.3-2ac58a2",
        "sequencerMetrics": {
          "gasPerSecond": 26090027.78,
          "forkChoiceUpdated": 0.0022,
          "getPayload": 0.0058,
          "sendTxs": 0.24
        },
        "validatorMetrics": {
          "gasPerSecond": 348413456.55,
          "newPayload": 0.067
        }
      },
      "thresholds": {
        "warning": {"sequencer/latency/get_payload": 1000000000},
        "error":   {"sequencer/latency/get_payload": 1500000000}
      },
      "createdAt": "2026-02-24T20:43:54.150038473Z",
      "machineInfo": {
        "type": "i7i.16xlarge",
        "provider": "",
        "region": "",
        "fileSystem": "ext4"
      }
    }
  ]
}
```

Every `<outputDir>/metadata.json` contains a **one-element `runs`
array**. The plural shape is preserved for forward-compat — the
report-api's parser accepts any length — but per-run files always
write exactly one element. Producers must not pack multiple runs
into one file in the new layout; that's a vestige of the legacy
central-`metadata/` directory.

#### Required fields

| Field | Type | Purpose |
|---|---|---|
| `runs[].id` | string | Unique run identifier. Used for deduplication. |
| `runs[].outputDir` | string | Path under the bucket where per-block metrics live. |
| `runs[].sourceFile` | string | Identifies the network. The report-api matches substrings (`mainnet`, `sepolia`, `testnet`, `devnet`) to scope comparison groups. |
| `runs[].testName` | string | Human-readable name shown in the report UI dropdown. |
| `runs[].testConfig.BenchmarkRun` | string | Cohort ID. All inner runs of one execution share this value. Drives the per-run page filter. |
| `runs[].createdAt` | RFC3339 timestamp | Used for retention, sort order, and time-bucket comparisons. |

Everything else is optional but recommended.

#### `testConfig` is the filter/comparison driver

The report's frontend **auto-discovers filter dropdowns** from any
`testConfig` key that has >1 distinct value across the active run
set. The report-api **auto-generates comparison groups** from
`testConfig.ClientVersion`. Two implications:

1. **Anything you want users to filter or compare on, write into
   `testConfig`** — including custom dimensions like `region`, `build`,
   `experiment`, `commit`.
2. **Avoid putting volatile values in `testConfig`** (timestamps,
   per-process state) — they'll create a dropdown entry per run and
   make the UI useless.

#### Standard `testConfig` keys

| Key | Meaning | Notes |
|---|---|---|
| `BenchmarkRun` | Cohort ID | Required. |
| `TransactionPayload` | Payload type | One per execution variant. |
| `GasLimit` | Block gas limit | int. |
| `BlockTimeMilliseconds` | Target block time | int. |
| `NodeType` | EL flavor under test | e.g., `builder`, `reth`, `geth`, `base-reth-node`. |
| `ClientVersion` | EL binary version | Format: `<name>/v<semver>-<7sha>`. The report-api groups by exact-match — pin it to a stable identifier per build. |
| `ValidatorNodeType` | Validator EL flavor | Optional; defaults to `NodeType`. |

You can add any other key. The UI handles them generically — no
frontend change required.

#### `ClientVersion` specifically

Today the Go runner captures the version by probing `<bin> --version`
after the EL starts up. The parser at
`runner/clients/common/version.go::ParseRethVersionOutput` handles
reth's multi-line format and returns `<version>-<7-char-sha>`.

Producers in other languages should write the same shape:
`<name>/v<semver>-<sha-prefix>`. Stable across rebuilds of the same
commit; different across different commits. If `ClientVersion` is
empty the run is silently excluded from version-comparison groups
(it still appears under its own `BenchmarkRun` ID).

The `BASE_BENCH_CLIENT_VERSION` env var beats the auto-detected
value — deployment scripts can pin a human-readable label like
`v1.2.3-rc1` instead of the raw build SHA.

### `<outputDir>/` directories

For each inner run, the runner writes per-block timeseries metrics
under a directory whose name matches `runs[].outputDir`:

```json
[
  {"BlockNumber": 1, "ExecutionMetrics": {"latency/get_payload": 4500000, ...}},
  {"BlockNumber": 2, "ExecutionMetrics": {...}},
  ...
]
```

One file per role (`metrics-sequencer.json`, `metrics-validator.json`).
The report-api serves them directly via
`GET /output/<outputDir>/metrics-<role>.json` — no merging,
no transformation. So the producer must write them in the final
on-the-wire shape.

The set of metric keys inside `ExecutionMetrics` is open. The
frontend has a registry of "known" metrics with units and labels at
`report/src/metricDefinitions.ts`; anything not listed there still
renders, just with raw key names.

## Comparison groups (automatic)

The report-api automatically synthesizes two kinds of comparison
"runs" alongside the natural ones:

- **`[Compare: Time] <network> <testName>`** — picks the latest run
  per variant per time bucket (`1d`/`1w`/`1m`).
- **`[Compare: Versions] <network> <testName>`** — picks the latest
  run per variant per distinct `ClientVersion`.

These appear in the existing dropdown next to the natural
`BenchmarkRun` IDs. Selecting one loads runs spanning the comparison
dimension; the user picks `Show Line Per: ClientVersion` (or any
other testConfig key) in the existing chart UI to overlay them as
separate chart series.

For these to populate:

- **Time comparison** needs runs spanning ≥2 time buckets. Any
  reasonable history satisfies this.
- **Version comparison** needs runs with ≥2 distinct
  `testConfig.ClientVersion` values. Empty/missing versions are
  silently skipped.

A cohort with fewer than 2 distinct buckets is dropped from the
dropdown — a "comparison" of one thing isn't a comparison.

## Producer responsibilities

A new producer (Rust, monorepo, whatever) needs to:

1. **Write one or more `metadata-<unique>.json` files** under
   `s3://<bucket>/metadata/`. Filename only matters in that the
   report-api lists everything ending in `.json` and processes it in
   listing order (later writes deduplicate earlier ones on the
   compound key `(id, outputDir)`).
2. **Write per-block metrics** to
   `s3://<bucket>/<outputDir>/metrics-<role>.json` for each
   `runs[].outputDir`. Use the role names the report knows
   (`sequencer`, `validator`) so the frontend's per-role panels
   populate, or document the custom role somewhere the user can find
   it.
3. **Populate `testConfig.ClientVersion`** with a value distinct per
   build. Without this, version-comparison groups can't form.
4. **Populate `createdAt`** as RFC3339. Required for retention and
   time-comparison.
5. Optionally populate `thresholds`, `machineInfo`, `testDescription`
   — improves the report UI but nothing breaks without them.

That's the whole contract. No registration, no schema service, no
central file to update.

## Migration from the legacy central-`metadata/` layout

Prior to the per-run cutover, the watcher uploaded the whole local
`metadata.json` (containing N inner runs) to a central
`metadata/metadata-<datetime>.json` directory on S3. The report-api
listed every file in that directory, parsed each, and deduplicated
runs by `(id, outputDir)`.

The per-run-directory layout is strictly better:

- **Co-location**: each run's metadata sits next to its metrics
  files. Deleting a run is one `aws s3 rm --recursive <outputDir>/`.
- **Atomic commit**: the producer writes `metadata.json` last, so an
  in-progress (or aborted) run is invisible until completion. No
  half-written metadata can ever reach the report-api.
- **Per-run isolation**: producers never write to a shared prefix.
- **No central file to update**: critical for the Rust runner — each
  inner run is a self-contained write-then-exit operation.

The `backend migrate` command is a one-shot script that splits any
remaining `metadata/metadata-*.json` files into per-run files and
optionally removes the legacy directory. See the cutover runbook in
the project NOTES for the exact sequence (pause the cron, run the
migration, deploy, resume).

## Long-term: drop the merger

A natural follow-on is to drop the merged `metadata.json` response
entirely and serve per-run JSON via an index endpoint
(`GET /api/v1/runs?since=<ts>` paginated). The frontend already has a
data-source abstraction in
`report/src/services/dataService.ts` — swapping out the metadata
fetch would be a single function change. But this requires a
frontend change, so it's a bigger lift than the producer-side
upgrades above.

The synthetic comparison groups in
`report-api/internal/services/comparison.go` are a function of the
full merged set, so dropping the merger means moving that synthesis
into the index endpoint's result. The data shape (a flat list of
runs, where some have synthetic IDs) stays identical from the
frontend's perspective.

## Frontend invariants (do not violate)

The report frontend depends on these invariants. Violate one and the
UI silently breaks.

1. **One `outputDir` per inner run.** The UI fetches metrics by
   `outputDir`; two runs sharing the same `outputDir` will fetch the
   same metrics file and display identical chart series.
2. **`BenchmarkRun` is a cohort key, not a run ID.** The per-run page
   filters `allRuns.filter(r => r.testConfig.BenchmarkRun === selected)`.
   Use `id` for per-run uniqueness; use `BenchmarkRun` to group runs
   that should be shown together.
3. **`testConfig` keys must be JSON-string-coercible.** The filter UI
   does string comparison on the values. Don't put objects or arrays
   there.
4. **`testName` should be stable per series.** The report-api adds
   prefixes for synthetic groups (`[Compare: …]`) and retention
   monthly survivors (`[Monthly - Mon YYYY] …`); the synthesizer
   strips these to canonicalize cohorts. If you add your own prefix
   convention, document it here and update
   `report-api/internal/services/comparison.go::canonicalTestName`
   so cohorts don't fragment.

## Pointers to code

- Report-api S3 read path & merger:
  `protocols/base-benchmarking/report-api/internal/services/s3.go`
- Comparison synthesizer:
  `protocols/base-benchmarking/report-api/internal/services/comparison.go`
- Frontend types (what the UI expects):
  `report/src/types.ts::BenchmarkRun`
- Frontend metric definitions (keys/units/labels):
  `report/src/metricDefinitions.ts`
- Frontend filter logic (how dropdowns are auto-discovered):
  `report/src/hooks/useBenchmarkFilters.ts` and `report/src/filter.ts`
- Current Go runner that produces this shape:
  `runner/benchmark/result_metadata.go::Run` and
  `runner/service.go::applyClientVersion`
