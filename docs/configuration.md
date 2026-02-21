# Configuration Guide

## Environment Variables

### Required for live CI features

| Variable | Required For | Description |
| --- | --- | --- |
| `ELECTRICITY_MAPS_API_KEY` | `suggest`, `run-aware`, `optimize`, `optimize-global`, `run --live-ci` | API key for Electricity Maps. |

### Optional in CI workflow

| Variable | Description |
| --- | --- |
| `CARBON_BUDGET_KG` | Budget gate threshold used by CI workflow. |
| `CARBON_BASELINE_KG` | Baseline emissions used for delta reporting. |

## Cache Configuration

Commands using forecast data support:

- `--cache-dir` (default: `~/.carbon-guard`)
- `--cache-ttl` (default: `10m`)

Example:

```bash
carbon-guard optimize \
  --zones DE,FR,PL \
  --duration 1800 \
  --lookahead 6 \
  --cache-dir ~/.carbon-guard \
  --cache-ttl 15m
```

## Timeout Configuration

`optimize` and `optimize-global` support:

- `--timeout` (default: `30s`)

Example:

```bash
carbon-guard optimize --zones DE,FR --duration 1200 --timeout 45s
```

## Budget/Baseline Conventions

- Keep budgets in `kgCO2`.
- Track a rolling baseline from recent successful CI runs.
- Fail on budget only after an initial observation period.

Recommended rollout:

1. Report-only mode for 1-2 weeks.
2. Set baseline from P50/P75 emissions.
3. Enable `--fail-on-budget` with a realistic threshold.
