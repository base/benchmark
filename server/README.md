# Report server

HTTP server that dynamically assembles the merged `GET /output/metadata.json`
response consumed by the React report UI in `report/`.

There is no central metadata file. The server reads individual
`<outputDir>/metadata.json` files from S3 on each request, merges them,
applies retention, and appends synthetic comparison groups. A two-level
cache (per-file by ETag + merged result with 1h TTL) keeps the hot path
to a single S3 `ListObjectsV2` call.

## Build

```bash
make build-server          # outputs ./bin/report-server
```

## Run locally

See `../local-stack/` in the `base-benchmarking-version-comparison` project for
a docker-compose setup that loads S3 data into MinIO and runs the server
against it.

```bash
export BASE_BENCH_API_S3_BUCKET=<bucket>
export BASE_BENCH_API_S3_ENDPOINT=http://localhost:9000  # MinIO only
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...

./bin/report-server
```

## Endpoints

| Endpoint | Description |
|---|---|
| `GET /output/metadata.json` | Merged + retained + comparison-synthesized run list |
| `GET /output/<outputDir>/metrics-<role>.json` | Per-block timeseries for one run |
| `GET /api/v1/load-tests/:network` | Load test run list |
| `GET /api/v1/load-tests/:network/:timestamp` | Single load test result |
| `GET /api/v1/health` | Health check |

## Data contract

See `docs/report-data-contract.md` for the S3 layout the server expects and
what producers must write to be visible in the report.
