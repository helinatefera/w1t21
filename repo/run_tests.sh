#!/bin/bash
# LedgerMint Test Runner
# Runs unit tests and API tests, prints PASS/FAIL summary
#
# Usage:
#   ./run_tests.sh              # Run all tests (API tests need services running)
#   ./run_tests.sh unit         # Run only unit tests
#   ./run_tests.sh api          # Run only API tests
#
# API tests require: docker compose up && seed data loaded

set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

TOTAL_PASS=0
TOTAL_FAIL=0
TOTAL_SKIP=0

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo ""
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}============================================${NC}"
    echo ""
}

print_result() {
    local name="$1" passed="$2" failed="$3"
    if [ "$failed" -eq 0 ]; then
        echo -e "  ${GREEN}PASS${NC} $name ($passed passed)"
    else
        echo -e "  ${RED}FAIL${NC} $name ($passed passed, $failed failed)"
    fi
}

# ==========================================
# UNIT TESTS (backend)
# ==========================================
run_go_tests() {
    print_header "UNIT TESTS"

    local go_pass=0
    local go_fail=0

    if [ -f "backend/go.mod" ]; then
        echo -e "${BLUE}Running unit tests from backend/...${NC}"
        output=$(cd backend && go test ./... -count=1 2>&1)
        exit_code=$?

        # Count passed/failed packages
        go_pass=$(echo "$output" | grep -c "^ok " || true)
        go_fail=$(echo "$output" | grep -c "^FAIL" || true)

        # Count individual test functions for a more accurate tally
        if [ $exit_code -eq 0 ]; then
            # Extract total test count from verbose-style or summary output
            test_count=$(cd backend && go test ./... -count=1 -v 2>&1 | grep -c "^--- PASS:" || true)
            go_pass=$((test_count))
        fi

        TOTAL_PASS=$((TOTAL_PASS + go_pass))
        TOTAL_FAIL=$((TOTAL_FAIL + go_fail))

        echo ""
        if [ $exit_code -eq 0 ]; then
            echo -e "  ${GREEN}Unit Tests: $go_pass passed, $go_fail failed${NC}"
        else
            echo -e "  ${RED}Unit Tests: $go_pass passed, $go_fail failed${NC}"
            echo "$output" | grep -E "^FAIL|FAIL\t|--- FAIL:" | head -10
        fi
    else
        echo -e "${YELLOW}WARNING: backend/go.mod not found, skipping unit tests${NC}"
        TOTAL_SKIP=$((TOTAL_SKIP + 1))
    fi
}

# ==========================================
# UNIT TESTS (frontend)
# ==========================================
run_unit_tests() {
    print_header "UNIT TESTS"

    local unit_pass=0
    local unit_fail=0

    # Install frontend deps (including devDependencies) before running Vitest.
    if [ -f "frontend/package.json" ] && grep -q '"vitest"' "frontend/package.json" 2>/dev/null; then
        local use_docker_node=0

        # Some environments use a Node patch version that lacks util.styleText.
        if command -v node >/dev/null 2>&1; then
            node -e "import('node:util').then(m=>process.exit(typeof m.styleText==='function'?0:1)).catch(()=>process.exit(1))" >/dev/null 2>&1 || use_docker_node=1
        else
            use_docker_node=1
        fi

        if [ "$use_docker_node" -eq 0 ]; then
            echo -e "${BLUE}Installing frontend dependencies (including dev)...${NC}"
            install_output=$(cd frontend && npm ci --include=dev 2>&1)
            install_exit_code=$?

            if [ $install_exit_code -ne 0 ]; then
                echo -e "  ${RED}Failed to install frontend dependencies${NC}"
                echo "$install_output" | tail -20
                TOTAL_FAIL=$((TOTAL_FAIL + 1))
                return
            fi

            echo -e "${BLUE}Running unit tests from frontend/...${NC}"
            output=$(cd frontend && npx vitest run --reporter=verbose --no-color 2>&1)
            exit_code=$?
        else
            if ! command -v docker >/dev/null 2>&1; then
                echo -e "  ${RED}Node runtime does not support styleText and Docker is unavailable for fallback${NC}"
                TOTAL_FAIL=$((TOTAL_FAIL + 1))
                return
            fi

            echo -e "${YELLOW}Local Node lacks styleText; running frontend unit tests in node:22-alpine...${NC}"
            output=$(docker run --rm \
                -v "$SCRIPT_DIR/frontend:/app" \
                -w /app \
                node:22-alpine \
                sh -lc "npm install --include=dev --no-package-lock >/dev/null 2>&1 && npx vitest run --reporter=verbose --no-color" 2>&1)
            exit_code=$?
        fi

        # Parse Vitest v4 summary line, e.g. "Tests  144 passed (144)"
        summary_line=$(echo "$output" | grep -E "^[[:space:]]*Tests[[:space:]]+" | tail -1 || true)
        total_passed=$(echo "$summary_line" | grep -oE "[0-9]+[[:space:]]+passed" | grep -oE "^[0-9]+" | head -1 || echo "0")
        total_failed=$(echo "$summary_line" | grep -oE "[0-9]+[[:space:]]+failed" | grep -oE "^[0-9]+" | head -1 || echo "0")

        total_passed=${total_passed:-0}
        total_failed=${total_failed:-0}

        if [ $exit_code -ne 0 ] && [ "$total_failed" -eq 0 ]; then
            total_failed=1
        fi

        unit_pass=$((total_passed))
        unit_fail=$((total_failed))

        TOTAL_PASS=$((TOTAL_PASS + unit_pass))
        TOTAL_FAIL=$((TOTAL_FAIL + unit_fail))

        echo ""
        if [ $exit_code -eq 0 ]; then
            echo -e "  ${GREEN}Unit Tests: $unit_pass passed, $unit_fail failed${NC}"
        else
            echo -e "  ${RED}Unit Tests: $unit_pass passed, $unit_fail failed${NC}"
            echo "$output" | grep -E "FAIL|AssertionError|Error" | head -10
        fi
    else
        echo -e "${YELLOW}WARNING: vitest not found in frontend/package.json, skipping unit tests${NC}"
        TOTAL_SKIP=$((TOTAL_SKIP + 1))
    fi
}

# ==========================================
# API TESTS
# ==========================================
run_api_tests() {
    print_header "API TESTS"

    local api_pass=0
    local api_fail=0

    # Check if services are running
    export API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"

    echo "Checking API availability at $API_BASE_URL..."
    if ! curl -s --max-time 5 "$API_BASE_URL/api/auth/login" > /dev/null 2>&1; then
        echo -e "${YELLOW}WARNING: API not reachable at $API_BASE_URL${NC}"
        echo "Make sure services are running: docker compose up"
        echo "And seed data is loaded: make seed"
        TOTAL_SKIP=$((TOTAL_SKIP + 1))
        return
    fi
    echo -e "${GREEN}API is reachable${NC}"

    # Enable test mode: disable rate limiting for API tests
    export DISABLE_RATE_LIMIT=true

    # Reset mutable application data so reruns start from a clean seeded state.
    # Keep roles, notification templates, and schema_migrations (reference/infra data).
    docker compose exec -T postgres psql -U ledgermint -d ledgermint -q <<'SQLRESET' >/dev/null 2>&1 || true
DO $$
DECLARE
    stmt text;
BEGIN
    SELECT 'TRUNCATE TABLE ' || string_agg(format('%I', tablename), ', ') || ' RESTART IDENTITY CASCADE;'
    INTO stmt
    FROM pg_tables
    WHERE schemaname = 'public'
      AND tablename NOT IN ('roles', 'notification_templates', 'schema_migrations');

    IF stmt IS NOT NULL THEN
        EXECUTE stmt;
    END IF;
END $$;
SQLRESET

    # Re-seed the database after truncation
    docker compose exec -T postgres psql -U ledgermint -d ledgermint -f /dev/stdin < scripts/seed.sql >/dev/null 2>&1 || true

    # Restart backend with DISABLE_RATE_LIMIT environment variable
    DISABLE_RATE_LIMIT=true docker compose up -d --no-deps backend > /dev/null 2>&1
    for _ in $(seq 1 30); do
        if curl -s --max-time 2 "$API_BASE_URL/api/auth/login" > /dev/null 2>&1; then
            break
        fi
        sleep 1
    done

    echo ""

    # Run setup check first — if it fails, abort remaining API tests
    if [ -f "API_tests/test_00_setup.sh" ]; then
        echo -e "${BLUE}Running test_00_setup (pre-flight checks)...${NC}"
        setup_output=$(bash "API_tests/test_00_setup.sh" 2>&1)
        setup_exit=$?
        setup_passed=$(echo "$setup_output" | grep -c "  PASS:")
        setup_failed=$(echo "$setup_output" | grep -c "  FAIL:")
        api_pass=$((api_pass + setup_passed))
        api_fail=$((api_fail + setup_failed))

        if [ "$setup_failed" -gt 0 ]; then
            print_result "test_00_setup" "$setup_passed" "$setup_failed"
            echo "$setup_output" | grep "  FAIL:" | head -10
            echo ""
            echo -e "${RED}Setup checks failed — skipping remaining API tests.${NC}"
            echo -e "${RED}Seed the database first:${NC}"
            echo -e "${RED}  docker compose exec postgres psql -U ledgermint -d ledgermint -f /dev/stdin < scripts/seed.sql${NC}"
            TOTAL_PASS=$((TOTAL_PASS + api_pass))
            TOTAL_FAIL=$((TOTAL_FAIL + api_fail))
            return
        else
            print_result "test_00_setup" "$setup_passed" "0"
        fi
        echo ""
    fi

    for test_file in API_tests/test_*.sh; do
        if [ ! -f "$test_file" ]; then
            continue
        fi
        # Skip setup — already ran above
        if [ "$(basename "$test_file")" = "test_00_setup.sh" ]; then
            continue
        fi
        test_name=$(basename "$test_file" .sh)
        echo -e "${BLUE}Running $test_name...${NC}"

        output=$(bash "$test_file" 2>&1)
        exit_code=$?

        # Extract pass/fail from output
        passed=$(echo "$output" | grep -c "  PASS:")
        failed=$(echo "$output" | grep -c "  FAIL:")

        api_pass=$((api_pass + passed))
        api_fail=$((api_fail + failed))

        if [ "$failed" -eq 0 ]; then
            print_result "$test_name" "$passed" "0"
        else
            print_result "$test_name" "$passed" "$failed"
            echo "$output" | grep "  FAIL:" | head -5
        fi
        echo ""

        # Clear rate limits between test files to prevent 429 cascade failures
        docker compose exec -T postgres psql -U ledgermint -d ledgermint -c "DELETE FROM login_attempts;" 2>/dev/null || true
        sleep 2
    done

    TOTAL_PASS=$((TOTAL_PASS + api_pass))
    TOTAL_FAIL=$((TOTAL_FAIL + api_fail))

    echo -e "  API Tests: ${api_pass} passed, ${api_fail} failed"
}

# ==========================================
# MAIN
# ==========================================
MODE="${1:-all}"

case "$MODE" in
    unit)
        run_go_tests
        run_unit_tests
        ;;
    go)
        run_go_tests
        ;;
    api)
        run_api_tests
        ;;
    all|*)
        run_go_tests
        run_unit_tests
        run_api_tests
        ;;
esac

# ==========================================
# FINAL SUMMARY
# ==========================================
print_header "TEST SUMMARY"

echo -e "  Total Passed:  ${GREEN}${TOTAL_PASS}${NC}"
echo -e "  Total Failed:  ${RED}${TOTAL_FAIL}${NC}"
if [ "$TOTAL_SKIP" -gt 0 ]; then
    echo -e "  Skipped:       ${YELLOW}${TOTAL_SKIP} suite(s)${NC}"
fi
echo ""

if [ "$TOTAL_FAIL" -eq 0 ] && [ "$TOTAL_PASS" -gt 0 ]; then
    echo -e "  ${GREEN}========================================${NC}"
    echo -e "  ${GREEN}  ALL TESTS PASSED ($TOTAL_PASS total)${NC}"
    echo -e "  ${GREEN}========================================${NC}"
    exit 0
elif [ "$TOTAL_FAIL" -gt 0 ]; then
    echo -e "  ${RED}========================================${NC}"
    echo -e "  ${RED}  SOME TESTS FAILED ($TOTAL_FAIL failures)${NC}"
    echo -e "  ${RED}========================================${NC}"
    exit 1
else
    echo -e "  ${YELLOW}========================================${NC}"
    echo -e "  ${YELLOW}  NO TESTS RAN${NC}"
    echo -e "  ${YELLOW}========================================${NC}"
    exit 1
fi