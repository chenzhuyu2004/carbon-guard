# Troubleshooting

## `carbon-guard execution failed`

Cause:
- Runtime binary execution failed in action container.

Checks:
1. Confirm workflow uses `uses: chenzhuyu2004/carbon-guard@v1`.
2. Open failed job logs and inspect the exact command error.
3. Ensure runtime inputs are provided (`duration` or `start_time`), or pass `github_token` for auto runtime detection.

## `Missing runtime input`

Cause:
- None of `duration`, `start_time`, `github_token`, `GH_ACTION_START_TIME` were provided.

Fix:
- Pass `with: duration`, or `with: start_time`, or `with: github_token: ${{ github.token }}` and grant `actions: read`.

## `Invalid duration`

Cause:
- Duration is not a positive integer.

Fix:
- Use seconds as integer values (for example `300`).

## `parse config file ...` or `read config file ...`

Cause:
- `--config` / `CARBON_GUARD_CONFIG` points to a missing or invalid JSON file.

Fix:
1. Verify path exists and is readable.
2. Use valid JSON and supported keys only:
   - `cache_dir`
   - `cache_ttl`
   - `timeout`
   - `output`
3. Run with `--config <path>` explicitly to confirm which file is being used.

## `missing ELECTRICITY_MAPS_API_KEY`

Cause:
- Live carbon intensity mode is enabled without API key.

Fix:
1. Add repository secret `ELECTRICITY_MAPS_API_KEY`.
2. Pass it to workflow env when using live CI endpoints.

## Budget gate failed

Cause:
- `emissions_kg > budget_kg` and `fail_on_budget=true`.

Fix:
1. Reduce CI duration.
2. Shift workload with `suggest` / `run-aware`.
3. Tune budget using historical baseline and realistic SLO.

## CI passes locally but fails on GitHub

Checks:
1. Verify `go test ./...`, `go vet ./...`, `go build` in clean checkout.
2. Confirm branch protection requires checks matching current workflow names.
3. Ensure allowed actions policy includes all referenced actions.
