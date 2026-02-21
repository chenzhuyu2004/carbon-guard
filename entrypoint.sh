#!/bin/sh
set -eu

fail() {
  echo "$1" >&2
  exit 1
}

is_non_negative_number() {
  echo "$1" | grep -Eq '^[0-9]+([.][0-9]+)?$'
}

float_gt() {
  awk -v a="$1" -v b="$2" 'BEGIN {exit !(a>b)}'
}

DURATION=""
BUDGET_EXCEEDED="false"
DELTA_VS_BASELINE_PCT=""

if [ -n "${INPUT_DURATION:-}" ]; then
  DURATION="$INPUT_DURATION"
elif [ -n "${INPUT_START_TIME:-}" ]; then
  NOW=$(date +%s)
  DURATION=$((NOW - INPUT_START_TIME))
elif [ -n "${GH_ACTION_START_TIME:-}" ]; then
  NOW=$(date +%s)
  DURATION=$((NOW - GH_ACTION_START_TIME))
else
  fail "Missing runtime input: provide either 'duration' or 'start_time' (or GH_ACTION_START_TIME for backward compatibility)."
fi

case "$DURATION" in
  ''|*[!0-9]*)
    fail "Invalid duration: must be a positive integer number of seconds."
    ;;
esac

if [ "$DURATION" -le 0 ]; then
  fail "Invalid duration: must be > 0 seconds."
fi

set -- --duration "$DURATION" --json

if [ -n "${INPUT_BUDGET_KG:-}" ]; then
  if ! is_non_negative_number "$INPUT_BUDGET_KG"; then
    fail "Invalid budget_kg: must be a non-negative number."
  fi
  set -- "$@" --budget-kg "$INPUT_BUDGET_KG"
fi

if [ -n "${INPUT_BASELINE_KG:-}" ]; then
  if ! is_non_negative_number "$INPUT_BASELINE_KG"; then
    fail "Invalid baseline_kg: must be a non-negative number."
  fi
  set -- "$@" --baseline-kg "$INPUT_BASELINE_KG"
fi

if ! /app/carbon-guard run "$@" > report.json; then
  fail "carbon-guard execution failed"
fi

EMISSIONS=$(awk -F: '/"emissions_kg"/ {gsub(/[ ,]/, "", $2); print $2; exit}' report.json)

if [ -z "$EMISSIONS" ]; then
  fail "Failed to parse emissions_kg from report.json"
fi

if [ -n "${INPUT_BUDGET_KG:-}" ]; then
  if float_gt "$EMISSIONS" "$INPUT_BUDGET_KG"; then
    BUDGET_EXCEEDED="true"
  fi
fi

if [ -n "${INPUT_BASELINE_KG:-}" ]; then
  DELTA_VS_BASELINE_PCT=$(awk -v e="$EMISSIONS" -v b="$INPUT_BASELINE_KG" 'BEGIN {if (b>0) printf "%.2f", ((e-b)/b*100)}')
fi

echo "emissions_kg=$EMISSIONS" >> "$GITHUB_OUTPUT"
echo "budget_exceeded=$BUDGET_EXCEEDED" >> "$GITHUB_OUTPUT"
echo "delta_vs_baseline_pct=$DELTA_VS_BASELINE_PCT" >> "$GITHUB_OUTPUT"

if [ "${INPUT_FAIL_ON_BUDGET:-false}" = "true" ] && [ "$BUDGET_EXCEEDED" = "true" ]; then
  fail "Carbon budget exceeded: emissions ${EMISSIONS}kg > budget ${INPUT_BUDGET_KG}kg."
fi
