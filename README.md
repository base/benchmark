# Base Benchmark Visualizer

Interactive, static visualizer for snapshot benchmark artifacts produced by
the benchmark runner in [`base/base`](https://github.com/base/base).

This repository **does not run benchmarks, start devnets, build execution
clients, or collect metrics**. Those responsibilities moved to the snapshot
benchmark runner in `base/base`. This project only renders the output bundle
that runner writes.

## Input data

The visualizer reads an output directory created by the `base/base` snapshot
benchmark runner. Point the local `output/` path at that directory before
starting the UI:

```text
output/
├── metadata.json
└── <benchmark-run>/
    ├── metadata.json
    ├── metrics-sequencer.json
    ├── metrics-validator.json
    ├── load-test-result.json
    └── validation.json
```

For example, use a symbolic link so results are never copied into this repo:

```bash
ln -s /absolute/path/to/benchmark-output output
```

`output/` is ignored by Git.

## Local development

Install the JavaScript dependencies and run Vite:

```bash
yarn install
yarn dev --host 0.0.0.0
```

Open the address printed by Vite. For a production-like static build:

```bash
yarn build
yarn preview --host 0.0.0.0
```

The build copies the local `output/` bundle into `dist/output/`, so the preview
server needs no API, Go service, or cloud credentials.

## Workflow

1. Run `base-bench snapshot` from the `base/base` repository and give that run
   a unique output directory.
2. Aggregate or select the resulting output directory as needed.
3. Link it to this repository as `output/`.
4. Use this UI to compare the sequencer- or validator-side metrics.

See the snapshot benchmark documentation in `base/base` for runner options and
artifact production. Changes to the artifact schema should be made there and
reflected in this visualizer's types and metric definitions.
