# GitHub Action Guide

Carbon Guard ships as a Docker-based GitHub Action.

## Inputs

| Name | Type | Required | Default | Description |
| --- | --- | --- | --- | --- |
| `duration` | string | No | `""` | Runtime in seconds. Highest priority if set. |
| `start_time` | string | No | `""` | Unix timestamp (seconds). Used when `duration` is empty. |
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

Carbon Guard resolves runtime inputs in this strict order:

1. `duration`
2. `start_time`
3. `GH_ACTION_START_TIME` (legacy backward-compatible env var)

If all are missing, the action fails fast.

## Minimal Example

```yaml
- name: Record start time
  run: echo "GH_ACTION_START_TIME=$(date +%s)" >> $GITHUB_ENV

- name: Carbon Guard
  id: carbon
  uses: chenzhuyu2004/carbon-guard@v1
  with:
    start_time: ${{ env.GH_ACTION_START_TIME }}

- name: Print emissions
  run: echo "emissions_kg=${{ steps.carbon.outputs.emissions_kg }}"
```

## Budget Gate Example

```yaml
- name: Carbon Guard (Budget Gate)
  uses: chenzhuyu2004/carbon-guard@v1
  with:
    start_time: ${{ env.GH_ACTION_START_TIME }}
    budget_kg: "0.0150"
    fail_on_budget: "true"
```

## Notes

- Keep budget and baseline as repository variables for consistency across workflows.
- For PR visibility, combine this action with a summary/comment step (see `docs/examples/`).
