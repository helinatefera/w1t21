#!/usr/bin/env bash


set -u -o pipefail


ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODE="${1:-all}"
RUN_ID="$$"
TEST_COMPOSE_FILE=".docker-compose.test.${RUN_ID}.yml"
TEST_COMPOSE_PATH="$ROOT_DIR/$TEST_COMPOSE_FILE"
COMPOSE_CMD=(docker compose -f "$TEST_COMPOSE_PATH")
COMPOSE_PROJECT_NAME="ledgermint_test_${RUN_ID}"


export COMPOSE_PROJECT_NAME


TOTAL_TESTS=0
TOTAL_PASSED=0
TOTAL_FAILED=0


UNIT_TOTAL=0
UNIT_PASSED=0
UNIT_FAILED=0


API_TOTAL=0
API_PASSED=0
API_FAILED=0


FAILED_TESTS=()


cleanup_requested=true


print_usage() {
 cat <<'EOF'
Usage: ./run_tests.sh [all|unit|api]


 all   Run unit tests and API tests (default)
 unit  Run only unit tests
 api   Run only API tests
EOF
}


compose() {
 (cd "$ROOT_DIR" && "${COMPOSE_CMD[@]}" "$@")
}


create_test_compose_file() {
 cat > "$TEST_COMPOSE_PATH" <<'EOF'
services:
 postgres:
   image: postgres:16-alpine
   environment:
     POSTGRES_DB: ledgermint
     POSTGRES_USER: ledgermint
     POSTGRES_PASSWORD: ${DB_PASSWORD:-changeme}
   healthcheck:
     test: ["CMD-SHELL", "pg_isready -U ledgermint"]
     interval: 5s
     retries: 10


 backend:
   build:
     context: ./backend
     dockerfile: Dockerfile
   depends_on:
     postgres:
       condition: service_healthy
   environment:
     APP_ENV: development
     DATABASE_URL: postgres://ledgermint:${DB_PASSWORD:-changeme}@postgres:5432/ledgermint?sslmode=disable
     JWT_SIGNING_KEY: ${JWT_SIGNING_KEY:-dev_jwt_secret_change_in_production}
     AES_MASTER_KEY: ${AES_MASTER_KEY:-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef}
     LISTEN_ADDR: ":8080"
     DISABLE_RATE_LIMIT: "true"


 tester:
   image: node:22-alpine
   working_dir: /workspace
   volumes:
     - ./:/workspace
   environment:
     API_BASE_URL: http://backend:8080
   entrypoint: ["sh", "-lc"]
EOF


 if [ ! -f "$TEST_COMPOSE_PATH" ]; then
   echo "ERROR: failed to create test compose file at $TEST_COMPOSE_PATH"
   return 1
 fi


 return 0
}


teardown_docker() {
 if [ "$cleanup_requested" != "true" ]; then
   return
 fi


 echo ""
 echo "--- Teardown: docker compose down -v ---"
 compose down -v --remove-orphans >/dev/null 2>&1 || true
 rm -f "$TEST_COMPOSE_PATH"
}


setup_docker() {
 if ! command -v docker >/dev/null 2>&1; then
   echo "ERROR: docker is not installed or not in PATH."
   return 1
 fi


 if ! create_test_compose_file; then
   return 1
 fi


 echo ""
 echo "--- Setup: resetting Docker state ---"
 compose down -v --remove-orphans >/dev/null 2>&1 || true


 echo "--- Setup: starting services (docker compose up -d) ---"
 if ! compose up -d --build postgres backend; then
   echo "WARN: first docker startup attempt failed, retrying with a clean reset..."
   compose down -v --remove-orphans >/dev/null 2>&1 || true
   if ! compose up -d --build postgres backend; then
     echo "ERROR: failed to start docker services."
     return 1
   fi
 fi


 echo "--- Setup: waiting for backend readiness ---"
 local tries=0
 local max_tries=40
 while [ "$tries" -lt "$max_tries" ]; do
   if compose run --rm --no-deps -T tester "apk add --no-cache curl >/dev/null && curl -s -o /dev/null -w '%{http_code}' http://backend:8080/api/setup/status | grep -qE '200|503'" >/dev/null 2>&1; then
     break
   fi
   tries=$((tries + 1))
   sleep 2
 done


 if [ "$tries" -ge "$max_tries" ]; then
   echo "ERROR: backend did not become ready in time."
   return 1
 fi


 echo "--- Setup: seeding database for API test state ---"
 if ! compose exec -T postgres sh -lc "psql -U ledgermint -d ledgermint -f /dev/stdin" < "$ROOT_DIR/scripts/seed.sql"; then
   echo "ERROR: failed to seed database."
   return 1
 fi


 return 0
}


reset_api_state() {
 echo "  -> Resetting API test state..."
 compose down -v --remove-orphans >/dev/null 2>&1 || true


 if ! compose up -d --build postgres backend >/dev/null; then
   echo "  -> ERROR: failed to restart services for retry"
   return 1
 fi


 local tries=0
 local max_tries=40
 while [ "$tries" -lt "$max_tries" ]; do
   if compose run --rm --no-deps -T tester "apk add --no-cache curl >/dev/null && curl -s -o /dev/null -w '%{http_code}' http://backend:8080/api/setup/status | grep -qE '200|503'" >/dev/null 2>&1; then
     break
   fi
   tries=$((tries + 1))
   sleep 2
 done


 if [ "$tries" -ge "$max_tries" ]; then
   echo "  -> ERROR: backend not ready after retry reset"
   return 1
 fi


 if ! compose exec -T postgres sh -lc "psql -U ledgermint -d ledgermint -f /dev/stdin" < "$ROOT_DIR/scripts/seed.sql" >/dev/null; then
   echo "  -> ERROR: failed to reseed database"
   return 1
 fi


 return 0
}


run_api_script_with_retry() {
 local script_name="$1"
 local cmd="apk add --no-cache bash curl python3 coreutils >/dev/null && bash /workspace/API_tests/$script_name"


 if compose run --rm --no-deps -T tester "$cmd"; then
   return 0
 fi


 echo "  -> First attempt failed for $script_name; retrying once with fresh state"
 if ! reset_api_state; then
   return 1
 fi


 compose run --rm --no-deps -T tester "$cmd"
}


run_test() {
 local suite="$1"
 local name="$2"
 shift 2


 TOTAL_TESTS=$((TOTAL_TESTS + 1))


 if [ "$suite" = "unit" ]; then
   UNIT_TOTAL=$((UNIT_TOTAL + 1))
 else
   API_TOTAL=$((API_TOTAL + 1))
 fi


 echo ""
 echo ">>> [$suite] Running $name"


 if "$@"; then
   echo ">>> [$suite] PASS - $name"
   TOTAL_PASSED=$((TOTAL_PASSED + 1))
   if [ "$suite" = "unit" ]; then
     UNIT_PASSED=$((UNIT_PASSED + 1))
   else
     API_PASSED=$((API_PASSED + 1))
   fi
 else
   echo ">>> [$suite] FAIL - $name"
   TOTAL_FAILED=$((TOTAL_FAILED + 1))
   if [ "$suite" = "unit" ]; then
     UNIT_FAILED=$((UNIT_FAILED + 1))
   else
     API_FAILED=$((API_FAILED + 1))
   fi
   FAILED_TESTS+=("$suite:$name")
 fi
}


run_unit_tests() {
 local unit_dir="$ROOT_DIR/unit_tests"
 local test_file
 local -a UNIT_FILES=()


 if [ ! -d "$unit_dir" ]; then
   echo "ERROR: unit test directory not found: $unit_dir"
   return 1
 fi


 for test_file in "$unit_dir"/*.test.js; do
   [ -f "$test_file" ] || continue
   UNIT_FILES+=("$test_file")
 done


 if [ "${#UNIT_FILES[@]}" -eq 0 ]; then
   echo "No unit tests found in $unit_dir"
   return 1
 fi


 echo "Preparing unit test dependencies inside Docker tester container..."
 if ! compose run --rm --no-deps -T tester "apk add --no-cache bash python3 >/dev/null && cd frontend && npm install --silent"; then
   echo "ERROR: failed to prepare unit test dependencies in container."
   return 1
 fi


 for test_file in "${UNIT_FILES[@]}"; do
   run_test "unit" "$(basename "$test_file")" compose run --rm --no-deps -T tester "apk add --no-cache bash python3 >/dev/null && cd frontend && npm run -s test -- ../unit_tests/$(basename "$test_file")"
 done
}


run_api_tests() {
 local api_dir="$ROOT_DIR/API_tests"
 local test_file
 local -a API_FILES=()


 if [ ! -d "$api_dir" ]; then
   echo "ERROR: API test directory not found: $api_dir"
   return 1
 fi


 for test_file in "$api_dir"/test_*.sh; do
   [ -f "$test_file" ] || continue
   API_FILES+=("$test_file")
 done


 if [ "${#API_FILES[@]}" -eq 0 ]; then
   echo "No API tests found in $api_dir"
   return 1
 fi


 for test_file in "${API_FILES[@]}"; do
   run_test "api" "$(basename "$test_file")" run_api_script_with_retry "$(basename "$test_file")"
 done
}


case "$MODE" in
 all|unit|api)
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


trap teardown_docker EXIT


echo "=================================================="
echo "LedgerMint Test Runner"
echo "Mode: $MODE"
echo "Root: $ROOT_DIR"
echo "Started: $(date '+%Y-%m-%d %H:%M:%S')"
echo "Compose project: $COMPOSE_PROJECT_NAME"
echo "=================================================="


if ! setup_docker; then
 TOTAL_FAILED=1
 FAILED_TESTS+=("setup:docker")
fi


if [ "$TOTAL_FAILED" -eq 0 ] && { [ "$MODE" = "all" ] || [ "$MODE" = "unit" ]; }; then
 echo ""
 echo "--- Running unit tests ---"
 run_unit_tests
fi


if [ "$TOTAL_FAILED" -eq 0 ] && { [ "$MODE" = "all" ] || [ "$MODE" = "api" ]; }; then
 echo ""
 echo "--- Running API tests ---"
 run_api_tests
fi


echo ""
echo "=================================================="
echo "Final Summary"
echo "=================================================="
if [ "$MODE" = "all" ] || [ "$MODE" = "unit" ]; then
 echo "Unit tests: total=$UNIT_TOTAL, passed=$UNIT_PASSED, failed=$UNIT_FAILED"
fi
if [ "$MODE" = "all" ] || [ "$MODE" = "api" ]; then
 echo "API tests : total=$API_TOTAL, passed=$API_PASSED, failed=$API_FAILED"
fi
echo "Overall   : total=$TOTAL_TESTS, passed=$TOTAL_PASSED, failed=$TOTAL_FAILED"


if [ "$TOTAL_FAILED" -gt 0 ]; then
 echo ""
 echo "Failed tests:"
 for test_name in "${FAILED_TESTS[@]}"; do
   echo "  - $test_name"
 done
 echo ""
 echo "RESULT: FAIL"
 exit 1
fi


echo ""
echo "RESULT: PASS"
exit 0




