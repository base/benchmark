# Local Results Layout

When running benchmarks locally, it helps to keep artifacts organized.

## Recommended directory layout
Create a top-level folder:
- `./local-results/`

Suggested structure:
- `local-results/<date>/<client>/<scenario>/`
Examples:
- `local-results/2025-01-12/geth/transfer-burst/`
- `local-results/2025-01-12/reth/state-growth/`

## What to store
- raw logs
- generated HTML reports
- config snapshots used for the run

## What NOT to store
- secrets
- RPC credentials
- private keys

## Tip
Add `local-results/` to `.gitignore` for safety.
