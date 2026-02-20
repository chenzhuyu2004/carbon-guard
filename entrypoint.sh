#!/bin/sh
set -e

if [ -n "$INPUT_DURATION" ]; then
  DURATION="$INPUT_DURATION"
else
  START=$(date +%s)
  sleep 1
  END=$(date +%s)
  DURATION=$((END - START))
fi

./carbon-guard run --duration "$DURATION" --json > report.json

EMISSIONS=$(grep -o '"emissions_kg":[ ]*[0-9.]*' report.json | grep -o '[0-9.]*')

echo "emissions_kg=$EMISSIONS" >> "$GITHUB_OUTPUT"
