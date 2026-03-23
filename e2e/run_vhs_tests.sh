#!/usr/bin/env bash
# E2E VHS tape tests for hystak.
# Runs all .tape files in e2e/tapes/ and verifies they complete successfully.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TAPES_DIR="${SCRIPT_DIR}/tapes"
ACTUAL_DIR="${SCRIPT_DIR}/actual"
BINARY="${HYSTAK_BINARY:-hystak}"

# Ensure binary is available
if ! command -v "$BINARY" &>/dev/null; then
    echo "ERROR: $BINARY not found in PATH"
    echo "Build with: go build -trimpath -o /usr/local/bin/hystak ."
    exit 1
fi

mkdir -p "${ACTUAL_DIR}"

# Create isolated config dir for E2E tests
export HYSTAK_CONFIG_DIR="$(mktemp -d)"
trap 'rm -rf "$HYSTAK_CONFIG_DIR"' EXIT

# Find tape files
tapes=("${TAPES_DIR}"/*.tape)
if [ ! -e "${tapes[0]}" ]; then
    echo "No .tape files found in ${TAPES_DIR} -- skipping E2E tests."
    exit 0
fi

passed=0
failed=0

for tape in "${tapes[@]}"; do
    name="$(basename "${tape}" .tape)"
    echo -n "Running tape: ${name} ... "
    if vhs "${tape}" -o "${ACTUAL_DIR}/${name}.gif" 2>"${ACTUAL_DIR}/${name}.log"; then
        echo "PASS"
        passed=$((passed + 1))
    else
        echo "FAIL"
        cat "${ACTUAL_DIR}/${name}.log" | sed 's/^/  /'
        failed=$((failed + 1))
    fi
done

echo ""
echo "Results: ${passed} passed, ${failed} failed"

if [ "${failed}" -gt 0 ]; then
    exit 1
fi
