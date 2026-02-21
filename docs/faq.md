# FAQ

## Why Carbon Guard?

CI emissions are usually invisible. Carbon Guard turns CI runtime into a measurable and enforceable signal.

## Is this only a reporting tool?

No. With `--budget-kg` and `--fail-on-budget`, it can actively gate merges/builds.

## How accurate is the result?

The model is deterministic and explicit. Accuracy depends on runtime duration, power profile assumptions, PUE, and CI source quality.

## Why no Cobra / external CLI framework?

The project intentionally stays standard-library-only for lower complexity, lower supply-chain risk, and better educational value.

## Can I use it without Electricity Maps API?

Yes. `run` supports static region factors and segment input. Live CI features require API access.

## Where should I tune budget and baseline?

Use GitHub repository variables:
- `CARBON_BUDGET_KG`
- `CARBON_BASELINE_KG`

Start from recent average emissions and tighten incrementally.
