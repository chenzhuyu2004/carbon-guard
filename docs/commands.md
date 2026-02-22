# CLI Command Reference

This page is the authoritative reference for all commands.

## Global Notes

- Use `--json` on `run` for machine-readable output.
- Use `--output text|json` on `optimize` and `optimize-global`.
- Commands using live carbon data require `ELECTRICITY_MAPS_API_KEY`.
- Shared defaults can be injected via config/env for `suggest`, `run-aware`, `optimize`, and `optimize-global`.
- Zone resolution supports `--zone-mode strict|fallback|auto`:
  - `strict`: zone(s) must be passed via CLI flag.
  - `fallback`: if CLI flag is empty, resolve from env (`CARBON_GUARD_ZONE` / `CARBON_GUARD_ZONES`) then config (`zone` / `zones`).
  - `auto`: fallback behavior plus auto hints (`CARBON_GUARD_ZONE_HINT` / `CARBON_GUARD_COUNTRY_HINT` / `CARBON_GUARD_TIMEZONE_HINT`) and locale/timezone heuristic (`LANG` / `LC_*` / `TZ`).

## `run`

Estimate emissions for a runtime.

### Syntax

```bash
carbon-guard run --duration <seconds> [flags]
```

### Flags

| Flag | Type | Default | Required | Description |
| --- | --- | --- | --- | --- |
| `--duration` | int | `0` | Yes | Runtime in seconds, must be `> 0`. |
| `--runner` | string | `ubuntu` | No | Runner profile: `ubuntu`, `windows`, `macos`. |
| `--region` | string | `global` | No | Static carbon-intensity region key. |
| `--load` | float | `0.6` | No | CPU load factor, range `[0,1]`. |
| `--pue` | float | `1.2` | No | Data center PUE, must be `>= 1.0`. |
| `--segments` | string | `""` | No | Dynamic CI segments: `duration:ci,duration:ci`. |
| `--live-ci` | string | `""` | No | Fetch live CI for a zone via API. |
| `--budget-kg` | float | `0` | No | Carbon budget in kgCO2. |
| `--baseline-kg` | float | `0` | No | Baseline emissions in kgCO2 for delta. |
| `--fail-on-budget` | bool | `false` | No | Return non-zero when emissions exceed budget. |
| `--json` | bool | `false` | No | Emit JSON output. |

### Examples

```bash
carbon-guard run --duration 300
carbon-guard run --duration 900 --runner windows --region us --load 0.8 --pue 1.25
carbon-guard run --duration 1200 --live-ci DE --json
carbon-guard run --duration 300 --budget-kg 0.01 --fail-on-budget
```

Text output auto-scales emissions across common units (`mg`, `g`, `kg`, `t`, `kt`, `Mt`, `Gt`) to keep the numeric value readable (target range `[1,1000)`), while still showing a `kgCO2` reference value.

## `suggest`

Recommend a lower-carbon execution window for one zone.

### Syntax

```bash
carbon-guard suggest --duration <seconds> [--zone <ZONE>] [flags]
```

### Flags

| Flag | Type | Default | Required | Description |
| --- | --- | --- | --- | --- |
| `--zone` | string | `""` | No | Electricity Maps zone (for example `DE`). Required when `--zone-mode strict` or env fallback is not set. |
| `--zone-mode` | string | `fallback` | No | Zone resolution mode: `strict`, `fallback`, or `auto` (`CLI > ENV > Config > Auto`). |
| `--duration` | int | `0` | Yes | Runtime in seconds. |
| `--threshold` | float | `0.35` | No | Current CI threshold (`kgCO2/kWh`). |
| `--lookahead` | int | `6` | No | Forecast lookahead in hours. |
| `--wait-cost` | float | `0` | No | Waiting penalty (`kgCO2/hour`) used in scheduling objective. |
| `--config` | string | `""` | No | Path to JSON config file for shared defaults. |
| `--cache-dir` | string | `~/.carbon-guard` | No | Forecast cache directory. |
| `--cache-ttl` | duration | `10m` | No | Cache TTL (Go duration format). |

## `run-aware`

Wait for greener conditions before running.

### Syntax

```bash
carbon-guard run-aware --duration <seconds> [--zone <ZONE>] [flags]
```

### Flags

| Flag | Type | Default | Required | Description |
| --- | --- | --- | --- | --- |
| `--zone` | string | `""` | No | Electricity Maps zone. Required when `--zone-mode strict` or env fallback is not set. |
| `--zone-mode` | string | `fallback` | No | Zone resolution mode: `strict`, `fallback`, or `auto` (`CLI > ENV > Config > Auto`). |
| `--duration` | int | `0` | Yes | Runtime in seconds. |
| `--threshold` | float | `0.35` | No | Legacy threshold used when `--threshold-enter/--threshold-exit` are unset. |
| `--threshold-enter` | float | `-1` | No | Run when current CI is `<= threshold-enter` (`kgCO2/kWh`). |
| `--threshold-exit` | float | `-1` | No | Keep waiting when current CI is `>= threshold-exit` (`kgCO2/kWh`). Must be `>= threshold-enter`. |
| `--lookahead` | int | `6` | No | Forecast lookahead in hours. |
| `--max-wait` | float | `6` | No | Maximum wait time in hours. |
| `--config` | string | `""` | No | Path to JSON config file for shared defaults. |
| `--cache-dir` | string | `~/.carbon-guard` | No | Forecast cache directory. |
| `--cache-ttl` | duration | `10m` | No | Cache TTL (Go duration format). |

## `optimize`

Compare multiple zones and rank by expected emission.

### Syntax

```bash
carbon-guard optimize --duration <seconds> [--zones <Z1,Z2,...>] [flags]
```

### Flags

| Flag | Type | Default | Required | Description |
| --- | --- | --- | --- | --- |
| `--zones` | string | `""` | No | Comma-separated zones, whitespace-safe. Required when `--zone-mode strict` or env fallback is not set. |
| `--zone-mode` | string | `fallback` | No | Zone resolution mode: `strict`, `fallback`, or `auto` (`CLI > ENV > Config > Auto`). |
| `--duration` | int | `0` | Yes | Runtime in seconds. |
| `--lookahead` | int | `6` | No | Forecast lookahead in hours. |
| `--wait-cost` | float | `0` | No | Waiting penalty (`kgCO2/hour`) used in zone ranking objective. |
| `--config` | string | `""` | No | Path to JSON config file for shared defaults. |
| `--timeout` | duration | `30s` | No | Command timeout (Go duration). |
| `--output` | string | `text` | No | `text` or `json`. |
| `--cache-dir` | string | `~/.carbon-guard` | No | Forecast cache directory. |
| `--cache-ttl` | duration | `10m` | No | Cache TTL. |

## `optimize-global`

Find the globally optimal `(time, zone)` over shared forecast timestamps.

### Syntax

```bash
carbon-guard optimize-global --duration <seconds> [--zones <Z1,Z2,...>] [flags]
```

### Flags

Same as `optimize`, including `--config`, plus:

| Flag | Type | Default | Required | Description |
| --- | --- | --- | --- | --- |
| `--resample-fill` | string | `forward` | No | Cross-zone resample fill mode: `forward` or `strict`. |
| `--resample-max-fill-age` | duration | `""` | No | Max forward-fill age. Empty means default `2*step` inferred from forecast cadence. Ignored when `--resample-fill strict`. |

When `--wait-cost > 0`, both `optimize` and `optimize-global` minimize:

`score = emission_kg + wait_cost * wait_hours`

## Exit Codes

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
