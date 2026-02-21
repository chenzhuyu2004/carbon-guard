# Carbon Guard

**The CLI that guilt-trips your CI pipeline into sustainability.**

![Go Report Card](https://img.shields.io/badge/go%20report-A%2B-brightgreen)
![License](https://img.shields.io/badge/license-MIT-blue)
![Carbon Savings](https://img.shields.io/badge/carbon%20savings-track%20it-success)

Carbon Guard helps engineering teams measure and reduce CI emissions with actionable, carbon-aware scheduling.

## Installation

### Go

```bash
go install github.com/czy/carbon-guard@latest
```

### Docker

```bash
docker run --rm ghcr.io/czy/carbon-guard:latest run --duration 300
```

## Key Features

- Real-time carbon intensity via Electricity Maps integration.
- **Run-aware scheduling** that waits for greener execution windows.
- **Global region optimization** to pick the cleanest zone for your workload.
- Built with **Clean Architecture (Hexagonal / Ports & Adapters)** for production-grade maintainability and learning.

## Architecture

```mermaid
graph TD
  CLI[CLI (cmd)] --> APP[App (Use Cases)]
  APP --> DOMAIN[Domain (Scheduling)]
  INFRA[Infrastructure (Electricity Maps)] --> DOMAIN
```

## Usage

```bash
# Basic emissions estimate
carbon-guard run --duration 300

# Enforce a carbon budget in CI
carbon-guard run --duration 300 --budget-kg 0.0100 --fail-on-budget

# Find the greenest zone and time window
carbon-guard optimize --zones "DE,FR,US-NY" --duration 3600

# Wait for a greener run window (max 2h)
carbon-guard run-aware --max-wait 2h
```

## GitHub Action Runtime Tracking

For accurate runtime emissions in GitHub Actions, pass either:
- `duration` directly, or
- `start_time` as a Unix timestamp (seconds).

If neither is provided, the action fails fast with a clear error.

```yaml
- name: Record start time
  run: echo "GH_ACTION_START_TIME=$(date +%s)" >> $GITHUB_ENV

- uses: czy/carbon-guard@v1
  with:
    start_time: ${{ env.GH_ACTION_START_TIME }}
    budget_kg: "0.2000"
    fail_on_budget: "true"
    baseline_kg: "0.1500"
  env:
    ELECTRICITY_MAPS_API_KEY: ${{ secrets.ELECTRICITY_MAPS_API_KEY }}
```

Backward compatibility: `GH_ACTION_START_TIME` environment variable is still supported.

Repository-level optimization:
- Set `CARBON_BUDGET_KG` and `CARBON_BASELINE_KG` as GitHub Repository Variables.
- CI will publish a Step Summary and post/update a Carbon Guard PR comment automatically.

## PR Comment Preview

```md
## Carbon Guard Report

- Duration: `612s`
- Emissions: `0.0138 kgCO2`
- Budget: `0.0150 kgCO2` (within budget)
- Baseline: `0.0164 kgCO2`
- Delta vs baseline: `-15.85%`
```

## Real-World Example (This Repository)

- Workflow runtime window analyzed: ~10 minutes
- Carbon budget threshold: `0.0150 kgCO2`
- Latest sample emission: `0.0138 kgCO2`
- Outcome: budget passed, with a double-digit reduction vs baseline

Use this as a template to set your own repository budget and baseline targets.

## Why Carbon Guard

CI is invisible infrastructure waste. Carbon Guard turns runtime into a carbon signal so teams can ship fast and smarter.
