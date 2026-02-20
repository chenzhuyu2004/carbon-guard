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

./carbon-guard run --duration $DURATION --json > report.json

EMISSIONS=$(cat report.json | sed -n 's/.*"emissions_kg":[ ]*\([0-9.]*\).*/\1/p')

echo "emissions_kg=$EMISSIONS" >> $GITHUB_OUTPUT
