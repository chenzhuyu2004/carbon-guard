# Optimization Roadmap

This file is the execution checklist for continuous improvement of Carbon Guard.

## Usage Rules

1. Pick items by priority order: `P0` first, then `P1`, then `P2`.
2. Move status: `TODO -> IN_PROGRESS -> DONE`.
3. For each `DONE`, append the merged PR link.
4. Every completed item must keep math model and CLI contract backward-compatible unless explicitly planned.

Status legend:

- `TODO`: not started
- `IN_PROGRESS`: actively implementing
- `DONE`: merged and released on `main`

## Baseline (Already Done)

| ID | Item | Priority | Status | Notes |
| --- | --- | --- | --- | --- |
| BASE-01 | Deterministic zone resolution chain `CLI > ENV > Config > Auto` | P0 | DONE | PR #33 |
| BASE-02 | Explicit auto hints (`zone_hint/country_hint/timezone_hint`) | P0 | DONE | PR #34 |

## Track A: Model & Algorithm

| ID | Item | Priority | Status | Acceptance Criteria |
| --- | --- | --- | --- | --- |
| ALG-01 | Replace heuristic country->zone fallback with curated provider zone mapping table | P0 | DONE | PR #35 |
| ALG-02 | Add CI data quality/confidence score propagation to scheduling decisions | P0 | TODO | Suggest/optimize outputs include CI confidence metadata and source completeness |
| ALG-03 | Add uncertainty-aware objective term (risk penalty) | P1 | TODO | Objective supports optional risk penalty without breaking existing default behavior |
| ALG-04 | Add budget-risk forecast (probability of budget exceedance in lookahead) | P1 | TODO | New output field in JSON mode, tested for deterministic scenarios |

## Track B: Scheduling Strategy

| ID | Item | Priority | Status | Acceptance Criteria |
| --- | --- | --- | --- | --- |
| SCH-01 | Add no-regret wait guard for `run-aware` (max acceptable delay for marginal gain) | P0 | DONE | Added `--max-delay-for-gain` and `--min-reduction-for-wait` for deterministic skip-on-marginal-gain behavior |
| SCH-02 | Support mixed forecast cadences (15m/30m/60m) with stable interpolation policy | P0 | TODO | `optimize-global` remains deterministic across mixed cadences |
| SCH-03 | Adaptive polling in `run-aware` (closer to best window => shorter poll interval) | P1 | TODO | Poll interval policy is explicit and bounded; no busy waiting |
| SCH-04 | Add what-if simulation mode for schedule strategy comparison | P2 | TODO | One command compares multiple policy presets over same forecast input |

## Track C: Reliability & Infra

| ID | Item | Priority | Status | Acceptance Criteria |
| --- | --- | --- | --- | --- |
| REL-01 | Standardize provider error taxonomy (auth/rate-limit/network/upstream/invalid-data) | P0 | DONE | `internal/ci.ProviderError` taxonomy with classification helpers and test coverage |
| REL-02 | Add circuit-breaker middleware around provider chain | P0 | TODO | Protect against repeated upstream failures; recovery is observable |
| REL-03 | Add stale-while-revalidate mode for forecast cache | P1 | TODO | Cached reads stay fast while refresh happens safely in background path |
| REL-04 | Export middleware metrics in machine-readable summary | P1 | TODO | Latency/retry/rate-limit/cache-hit counters available in CI output |
| REL-05 | Optional distributed cache lock mode for shared runners | P2 | TODO | Documented constraints for NFS/shared volumes and locking behavior |

## Track D: Product Output & UX

| ID | Item | Priority | Status | Acceptance Criteria |
| --- | --- | --- | --- | --- |
| UX-01 | Add JSON schema version field for all machine outputs | P0 | DONE | `schema_version` now present in `run` JSON, optimize JSON, optimize-global JSON, and JSON error output |
| UX-02 | Add output verbosity levels (`minimal/standard/debug`) | P0 | TODO | Text output can be concise for CI logs and detailed for diagnostics |
| UX-03 | Add deterministic locale/timezone formatting policy in all text outputs | P1 | TODO | Time/unit formatting is explicit and test-covered |
| UX-04 | Add built-in PR-comment markdown generator mode | P1 | TODO | CI can publish consistent summary without custom scripts |

## Track E: Repo & Operations

| ID | Item | Priority | Status | Acceptance Criteria |
| --- | --- | --- | --- | --- |
| OPS-01 | Auto-generate command docs from actual flags | P0 | TODO | `docs/commands.md` drift is eliminated |
| OPS-02 | Add compatibility matrix (`Go`, `Runner`, `Action`) | P0 | TODO | Explicit support matrix in docs and CI |
| OPS-03 | Add benchmark suite for scheduling core | P1 | TODO | Track complexity/runtime before and after optimization changes |
| OPS-04 | Add security threat model document for provider/cache/action surfaces | P1 | TODO | Security assumptions and mitigations are explicit and reviewable |

## Next 3 Milestones

| Milestone | Scope | Target |
| --- | --- | --- |
| M1 | ALG-01, REL-01, UX-01 | Completed on main branch + v1.0.4 line |
| M2 | SCH-01, SCH-02, REL-02 | Improve scheduling quality and failure resilience |
| M3 | OPS-01, UX-04, REL-04 | Improve maintainability and CI product experience |
