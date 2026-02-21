# CLI Command Reference

## `run`

Estimate emissions for a runtime.

Example:

```bash
carbon-guard run --duration 300 --json
```

Key flags:

- `--duration` required, runtime seconds.
- `--budget-kg` optional carbon budget.
- `--baseline-kg` optional baseline for percentage delta.
- `--fail-on-budget` exit non-zero when budget exceeded.
- `--json` machine-readable output.

## `suggest`

Recommend a greener execution window for one zone forecast.

Example:

```bash
carbon-guard suggest --zone DE --duration 1800 --lookahead 6
```

## `run-aware`

Wait for better carbon conditions, then run.

Example:

```bash
carbon-guard run-aware --zone DE --threshold 0.35 --max-wait 2h
```

## `optimize`

Compare zones and rank by expected emissions.

Example:

```bash
carbon-guard optimize --zones DE,FR,PL --duration 1800 --lookahead 6
```

## `optimize-global`

Find best `(time, zone)` pair over common forecast timestamps.

Example:

```bash
carbon-guard optimize-global --zones DE,FR,PL --duration 1800 --lookahead 6
```

## Standard Exit Codes

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Input error |
| `2` | Provider/API error |
| `10` | Max wait exceeded |
| `11` | Missed optimal window |
| `12` | Timeout |
| `20` | No valid window found |
| `21` | Budget exceeded |
