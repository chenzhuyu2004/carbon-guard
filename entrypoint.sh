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

resolve_duration_from_run_metadata() {
  token="${INPUT_GITHUB_TOKEN:-${GITHUB_TOKEN:-}}"
  if [ -z "$token" ] || [ -z "${GITHUB_REPOSITORY:-}" ] || [ -z "${GITHUB_RUN_ID:-}" ]; then
    return 1
  fi

  api_url="https://api.github.com/repos/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}"
  run_json=""

  if command -v wget >/dev/null 2>&1; then
    run_json=$(wget -qO- \
      --header="Authorization: Bearer ${token}" \
      --header="Accept: application/vnd.github+json" \
      "$api_url" 2>/dev/null || true)
  elif command -v curl >/dev/null 2>&1; then
    run_json=$(curl -fsSL \
      -H "Authorization: Bearer ${token}" \
      -H "Accept: application/vnd.github+json" \
      "$api_url" 2>/dev/null || true)
  fi

  [ -z "$run_json" ] && return 1

  run_started_at=$(echo "$run_json" | tr -d '\n' | sed -n 's/.*"run_started_at":"\([^"]*\)".*/\1/p')
  [ -z "$run_started_at" ] && return 1

  start_epoch=$(date -D '%Y-%m-%dT%H:%M:%SZ' -d "$run_started_at" +%s 2>/dev/null || date -u -d "$run_started_at" +%s 2>/dev/null || true)
  [ -z "$start_epoch" ] && return 1

  now_epoch=$(date +%s)
  runtime=$((now_epoch - start_epoch))
  [ "$runtime" -le 0 ] && return 1

  DURATION="$runtime"
  return 0
}

if [ -n "${INPUT_DURATION:-}" ]; then
  DURATION="$INPUT_DURATION"
elif [ -n "${INPUT_START_TIME:-}" ]; then
  NOW=$(date +%s)
  DURATION=$((NOW - INPUT_START_TIME))
elif resolve_duration_from_run_metadata; then
  :
elif [ -n "${GH_ACTION_START_TIME:-}" ]; then
  NOW=$(date +%s)
  DURATION=$((NOW - GH_ACTION_START_TIME))
else
  fail "Missing runtime input: provide duration/start_time, or github_token for auto runtime detection."
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
