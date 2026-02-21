# GitHub Action Guide

Carbon Guard ships as a Docker-based GitHub Action.

## Inputs

| Name | Type | Required | Default | Description |
| --- | --- | --- | --- | --- |
| `duration` | string | No | `""` | Runtime in seconds. Highest priority if set. |
| `start_time` | string | No | `""` | Unix timestamp (seconds). Used when `duration` is empty. |
| `github_token` | string | No | `""` | Token for auto runtime detection from current workflow run metadata. |
| `budget_kg` | string | No | `""` | Optional carbon budget in kgCO2. |
| `fail_on_budget` | string | No | `"false"` | If `"true"`, action fails when `emissions_kg > budget_kg`. |
| `baseline_kg` | string | No | `""` | Optional baseline in kgCO2 for delta calculation. |

## Outputs

| Name | Type | Description |
| --- | --- | --- |
| `emissions_kg` | string | Estimated emissions in kgCO2. |
| `budget_exceeded` | string | `true` or `false` when budget is provided. |
| `delta_vs_baseline_pct` | string | Percentage delta vs baseline (empty if no baseline). |

## Runtime Resolution Order

Carbon Guard resolves runtime inputs in this order:

1. `duration`
2. `start_time`
3. `github_token` (or `GITHUB_TOKEN`) via GitHub Actions run metadata
4. `GH_ACTION_START_TIME` (legacy backward-compatible env var)

If all are missing, the action fails fast.

## Simplest Example

```yaml
- name: Carbon Guard
  id: carbon
  uses: chenzhuyu2004/carbon-guard@v1
  with:
    duration: "300"

- name: Print emissions
  run: echo "emissions_kg=${{ steps.carbon.outputs.emissions_kg }}"
```

## Real Runtime Without Extra Start-Time Step

```yaml
permissions:
  contents: read
  actions: read

jobs:
  carbon:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Carbon Guard
        id: carbon
        uses: chenzhuyu2004/carbon-guard@v1
        with:
          github_token: ${{ github.token }}
```

## Budget Gate Example

```yaml
permissions:
  contents: read
  actions: read

- name: Carbon Guard (Budget Gate)
  uses: chenzhuyu2004/carbon-guard@v1
  with:
    github_token: ${{ github.token }}
    budget_kg: "0.0150"
    fail_on_budget: "true"
```

## Notes

- Keep budget and baseline as repository variables for consistency across workflows.
- For PR visibility, combine this action with a summary/comment step (see `docs/examples/`).
