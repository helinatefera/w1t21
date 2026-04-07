#!/usr/bin/env bash

set -u -o pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODE="${1:-unit}"

TOTAL_TESTS=0
TOTAL_PASSED=0
TOTAL_FAILED=0
BACKEND_TOTAL=0
BACKEND_PASSED=0
BACKEND_FAILED=0
FRONTEND_TOTAL=0
FRONTEND_PASSED=0
FRONTEND_FAILED=0

print_usage() {
  cat <<'EOF'
Usage: ./run_tests.sh [unit|go|frontend|node]

unit      Run backend Go tests and frontend Node/Vitest tests (default)
go        Run only backend Go tests
frontend  Run only frontend Node/Vitest tests
node      Alias for frontend
EOF
}

run_backend_tests() {
  local backend_dir="$ROOT_DIR/backend"
  local output=""
  local exit_code=0
  local passed=0
  local failed=0

  if [ ! -f "$backend_dir/go.mod" ]; then
    echo "ERROR: backend/go.mod not found"
    BACKEND_TOTAL=$((BACKEND_TOTAL + 1))
    BACKEND_FAILED=$((BACKEND_FAILED + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    TOTAL_FAILED=$((TOTAL_FAILED + 1))
    return 1
  fi

  echo ""
  echo "--- Running backend Go tests ---"

  output=$(cd "$backend_dir" && go test ./... -count=1 2>&1)
  exit_code=$?

  passed=$(printf '%s
' "$output" | grep -c '^ok ' || true)
  failed=$(printf '%s
' "$output" | grep -c '^FAIL' || true)

  if [ "$exit_code" -eq 0 ] && [ "$passed" -eq 0 ]; then
    passed=1
  fi

  if [ "$exit_code" -ne 0 ] && [ "$failed" -eq 0 ]; then
    failed=1
  fi

  BACKEND_TOTAL=$((BACKEND_TOTAL + passed + failed))
  BACKEND_PASSED=$((BACKEND_PASSED + passed))
  BACKEND_FAILED=$((BACKEND_FAILED + failed))
  TOTAL_TESTS=$((TOTAL_TESTS + passed + failed))
  TOTAL_PASSED=$((TOTAL_PASSED + passed))
  TOTAL_FAILED=$((TOTAL_FAILED + failed))

  if [ "$exit_code" -eq 0 ]; then
    echo "Backend Go tests: passed=$passed failed=$failed"
    return 0
  fi

  echo "Backend Go tests failed"
  printf '%s
' "$output" | grep -E '^FAIL|--- FAIL:' || true
  return 1
}

run_frontend_tests() {
  local frontend_dir="$ROOT_DIR/frontend"
  local output=""
  local exit_code=0
  local passed=0
  local failed=0
  local install_cmd=""

  if [ ! -f "$frontend_dir/package.json" ]; then
    echo "ERROR: frontend/package.json not found"
    FRONTEND_TOTAL=$((FRONTEND_TOTAL + 1))
    FRONTEND_FAILED=$((FRONTEND_FAILED + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    TOTAL_FAILED=$((TOTAL_FAILED + 1))
    return 1
  fi

  if ! grep -q '"vitest"' "$frontend_dir/package.json" 2>/dev/null; then
    echo "ERROR: vitest is not declared in frontend/package.json"
    FRONTEND_TOTAL=$((FRONTEND_TOTAL + 1))
    FRONTEND_FAILED=$((FRONTEND_FAILED + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    TOTAL_FAILED=$((TOTAL_FAILED + 1))
    return 1
  fi

  echo ""
  echo "--- Preparing frontend dependencies ---"
  if [ -f "$frontend_dir/package-lock.json" ]; then
    install_cmd="npm ci --include=dev"
  else
    install_cmd="npm install --include=dev"
  fi

  if ! (cd "$frontend_dir" && sh -lc "$install_cmd" >/dev/null 2>&1); then
    echo "ERROR: failed to install frontend dependencies"
    FRONTEND_TOTAL=$((FRONTEND_TOTAL + 1))
    FRONTEND_FAILED=$((FRONTEND_FAILED + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    TOTAL_FAILED=$((TOTAL_FAILED + 1))
    return 1
  fi

  echo "--- Running frontend Node/Vitest tests ---"
  output=$(cd "$frontend_dir" && npx vitest run --reporter=verbose --no-color 2>&1)
  exit_code=$?

  if [ "$exit_code" -eq 0 ]; then
    passed=$(printf '%s
' "$output" | grep -E '^[[:space:]]*Tests[[:space:]]+' | tail -1 | grep -oE '[0-9]+[[:space:]]+passed' | grep -oE '^[0-9]+' | head -1 || echo "0")
    failed=$(printf '%s
' "$output" | grep -E '^[[:space:]]*Tests[[:space:]]+' | tail -1 | grep -oE '[0-9]+[[:space:]]+failed' | grep -oE '^[0-9]+' | head -1 || echo "0")
    passed=${passed:-0}
    failed=${failed:-0}
    if [ "$passed" -eq 0 ] && [ "$failed" -eq 0 ]; then
      passed=1
    fi
  else
    failed=1
  fi

  FRONTEND_TOTAL=$((FRONTEND_TOTAL + passed + failed))
  FRONTEND_PASSED=$((FRONTEND_PASSED + passed))
  FRONTEND_FAILED=$((FRONTEND_FAILED + failed))
  TOTAL_TESTS=$((TOTAL_TESTS + passed + failed))
  TOTAL_PASSED=$((TOTAL_PASSED + passed))
  TOTAL_FAILED=$((TOTAL_FAILED + failed))

  if [ "$exit_code" -eq 0 ]; then
    echo "Frontend tests: passed=$passed failed=$failed"
    return 0
  fi

  echo "Frontend tests failed"
  printf '%s
' "$output" | grep -E 'FAIL|AssertionError|Error' || true
  return 1
}

case "$MODE" in
  unit|all)
    ;;
  go|frontend|node)
    ;;
  -h|--help|help)
    print_usage
    exit 0
    ;;
  *)
    echo "ERROR: Unknown mode '$MODE'"
    echo ""
    print_usage
    exit 2
    ;;
esac

cd "$ROOT_DIR"

echo "=================================================="
echo "LedgerMint Test Runner"
echo "Mode: $MODE"
echo "Root: $ROOT_DIR"
echo "Started: $(date '+%Y-%m-%d %H:%M:%S')"
echo "=================================================="

if [ "$MODE" = "unit" ] || [ "$MODE" = "all" ]; then
  run_backend_tests
  run_frontend_tests
elif [ "$MODE" = "go" ]; then
  run_backend_tests
elif [ "$MODE" = "frontend" ] || [ "$MODE" = "node" ]; then
  run_frontend_tests
fi

echo ""
echo "=================================================="
echo "Final Summary"
echo "=================================================="
echo "Backend tests : total=$BACKEND_TOTAL, passed=$BACKEND_PASSED, failed=$BACKEND_FAILED"
echo "Frontend tests: total=$FRONTEND_TOTAL, passed=$FRONTEND_PASSED, failed=$FRONTEND_FAILED"
echo "Overall       : total=$TOTAL_TESTS, passed=$TOTAL_PASSED, failed=$TOTAL_FAILED"

if [ "$TOTAL_FAILED" -gt 0 ]; then
  echo ""
  echo "RESULT: FAIL"
  exit 1
fi

echo ""
echo "RESULT: PASS"
exit 0
