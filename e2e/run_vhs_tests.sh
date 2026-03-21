#!/usr/bin/env bash
# E2E test runner for hystak using VHS tape scripts.
#
# Usage:
#   bash e2e/run_vhs_tests.sh          # run all tape tests
#   bash e2e/run_vhs_tests.sh --update  # regenerate golden files
#
# Prerequisites: vhs, go
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY="/tmp/hystak-e2e-test"
UPDATE=false

if [[ "${1:-}" == "--update" ]]; then
  UPDATE=true
fi

# Build binary
echo "==> Building hystak..."
(cd "$PROJECT_ROOT" && go build -trimpath -o "$BINARY" .)
export PATH="$(dirname "$BINARY"):$PATH"

# Create output directory
mkdir -p "$SCRIPT_DIR/actual"

FAILED=0
PASSED=0

run_tape() {
  local tape="$1"
  local name
  name="$(basename "$tape" .tape)"

  echo "==> Running tape: $name"

  # Create isolated config directory
  local config_dir
  config_dir="$(mktemp -d)"
  cp "$SCRIPT_DIR/fixtures/"*.yaml "$config_dir/"

  # Create project directory referenced in fixtures
  mkdir -p /tmp/hystak-e2e-demo

  export HYSTAK_CONFIG_DIR="$config_dir"

  # Run VHS
  if ! (cd "$PROJECT_ROOT" && vhs "$tape" 2>/dev/null); then
    echo "  FAIL: VHS execution failed for $name"
    FAILED=$((FAILED + 1))
    rm -rf "$config_dir"
    return
  fi

  # For .txt output, compare against golden files
  local actual="$SCRIPT_DIR/actual/${name}.txt"
  local golden="$SCRIPT_DIR/golden/${name}.txt"

  if [[ -f "$actual" ]]; then
    if [[ "$UPDATE" == "true" ]]; then
      mkdir -p "$SCRIPT_DIR/golden"
      cp "$actual" "$golden"
      echo "  UPDATED: $golden"
      PASSED=$((PASSED + 1))
    elif [[ -f "$golden" ]]; then
      if diff -u "$golden" "$actual" > /dev/null 2>&1; then
        echo "  PASS: $name"
        PASSED=$((PASSED + 1))
      else
        echo "  FAIL: output differs for $name"
        diff -u "$golden" "$actual" || true
        FAILED=$((FAILED + 1))
      fi
    else
      echo "  SKIP: no golden file for $name (run with --update to create)"
      PASSED=$((PASSED + 1))
    fi
  else
    # GIF-only tape (smoke test — pass if VHS succeeded)
    echo "  PASS: $name (smoke test)"
    PASSED=$((PASSED + 1))
  fi

  rm -rf "$config_dir"
}

# Run all tapes
for tape in "$SCRIPT_DIR"/tapes/*.tape; do
  run_tape "$tape"
done

# Summary
echo ""
echo "==> Results: $PASSED passed, $FAILED failed"

if [[ "$FAILED" -gt 0 ]]; then
  exit 1
fi
