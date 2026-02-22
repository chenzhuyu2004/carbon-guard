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

### Optional shared CLI defaults

These apply to `suggest`, `run-aware`, `optimize`, and `optimize-global`.

| Variable | Description |
| --- | --- |
| `CARBON_GUARD_CONFIG` | Path to JSON config file. |
| `CARBON_GUARD_CACHE_DIR` | Default cache directory. |
| `CARBON_GUARD_CACHE_TTL` | Default cache TTL (Go duration). |
| `CARBON_GUARD_TIMEOUT` | Default timeout (Go duration). |
| `CARBON_GUARD_OUTPUT` | Default output mode (`text` or `json`). |
| `CARBON_GUARD_ZONE` | Default single zone fallback for `suggest` / `run-aware`. |
| `CARBON_GUARD_ZONES` | Default zone list fallback for `optimize` / `optimize-global`. |

## Config File (JSON)

Use `--config <path>` or `CARBON_GUARD_CONFIG`.

Example:

```json
{
  "cache_dir": "~/.carbon-guard",
  "cache_ttl": "15m",
  "timeout": "45s",
  "output": "json"
}
```

Supported keys:

- `cache_dir`
- `cache_ttl`
- `timeout`
- `output`

## Precedence Rules

Shared defaults resolve in this order:

1. CLI flags
2. Environment variables
3. Config file
4. Built-in defaults

## Cache Configuration

Commands using forecast data support:

- `--cache-dir` (default: `~/.carbon-guard`)
- `--cache-ttl` (default: `10m`)
- `--config` (optional JSON defaults)

`run-aware` also supports hysteresis thresholds:

- `--threshold-enter`
- `--threshold-exit`

If both are unset, `--threshold` (legacy flag) is used for backward compatibility.

## Zone Resolution Strategy

Commands with zone inputs support:

- `--zone-mode strict|fallback|auto` (default: `fallback`)

Behavior:

1. `strict`: zone(s) must be provided via CLI (`--zone` / `--zones`).
2. `fallback`: if CLI zone flags are empty, commands use `CARBON_GUARD_ZONE` / `CARBON_GUARD_ZONES`.
3. `auto`: run fallback first, then attempt locale/timezone inference using `LC_ALL`, `LC_MESSAGES`, `LANG`, and `TZ`.

Examples:

```bash
CARBON_GUARD_ZONE=DE carbon-guard suggest --duration 1200
CARBON_GUARD_ZONES=DE,FR carbon-guard optimize --duration 1800
carbon-guard run-aware --zone-mode strict --zone US-NY --duration 900
LANG=de_DE.UTF-8 carbon-guard suggest --zone-mode auto --duration 900
```

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
- `--config` (optional JSON defaults)

`optimize-global` additionally supports:

- `--resample-fill` (`forward|strict`, default `forward`)
- `--resample-max-fill-age` (Go duration, empty means default `2*step`)

Example:

```bash
carbon-guard optimize --zones DE,FR --duration 1200 --timeout 45s
carbon-guard optimize-global --zones DE,FR,PL --duration 1800 --resample-fill strict
```

## Scheduling Objective

`suggest`, `optimize`, and `optimize-global` support:

- `--wait-cost` (default: `0`)

Unit is `kgCO2/hour`. The optimizer minimizes:

`score = emission_kg + wait_cost * wait_hours`

Setting `--wait-cost 0` keeps pure-emission optimization behavior.

## Budget/Baseline Conventions

- Keep budgets in `kgCO2`.
- Track a rolling baseline from recent successful CI runs.
- Fail on budget only after an initial observation period.

Recommended rollout:

1. Report-only mode for 1-2 weeks.
2. Set baseline from P50/P75 emissions.
3. Enable `--fail-on-budget` with a realistic threshold.
