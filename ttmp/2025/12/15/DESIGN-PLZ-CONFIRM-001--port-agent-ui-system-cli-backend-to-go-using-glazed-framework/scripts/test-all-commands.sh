#!/bin/bash
set -e

# Colors for output
GREEN='\033[1;32m'
BLUE='\033[1;34m'
YELLOW='\033[1;33m'
RED='\033[1;31m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TICKET_DIR="$(dirname "$SCRIPT_DIR")"
# Find repo root by looking for go.mod (go up from scripts/ -> ticket/ -> ttmp/2025/12/15/ -> ttmp/2025/12/ -> ttmp/2025/ -> ttmp/ -> plz-confirm/)
REPO_ROOT="$(cd "$SCRIPT_DIR" && while [ ! -f go.mod ] && [ "$PWD" != "/" ]; do cd ..; done && pwd)"
if [ ! -f "$REPO_ROOT/go.mod" ]; then
    # Fallback: assume we're in plz-confirm repo
    REPO_ROOT="$(cd "$SCRIPT_DIR/../../../../.." && pwd)"
    if [ ! -f "$REPO_ROOT/go.mod" ]; then
        echo "Error: Could not find go.mod (repo root)" >&2
        exit 1
    fi
fi

print_header() {
    echo ""
    echo -e "${GREEN}=== $1 ===${NC}"
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WAIT]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if server is running
if ! curl -s http://localhost:3001/api/requests > /dev/null 2>&1; then
    print_error "Backend server not running on :3001"
    print_info "Start it with: cd $REPO_ROOT && go run ./cmd/plz-confirm serve"
    exit 1
fi

# Check if Vite is running
if ! curl -s http://localhost:3000 > /dev/null 2>&1; then
    print_warning "Vite dev server not running on :3000 (may still work if backend is accessible)"
fi

# Parse arguments: if a number is provided, run only that test
TEST_NUM=""
if [ $# -gt 0 ]; then
    if [[ "$1" =~ ^[1-5]$ ]]; then
        TEST_NUM="$1"
        print_info "Running only test $TEST_NUM"
    else
        print_error "Invalid test number: $1 (must be 1-5)"
        exit 1
    fi
fi

print_header "Agent UI System - Full Command Test Suite"
print_info "Make sure the web UI is open at http://localhost:3000"
if [ -z "$TEST_NUM" ]; then
    print_info "This script will run all widget commands sequentially"
else
    print_info "Running single test: $TEST_NUM"
fi
echo ""
if [ -t 0 ]; then
    # Interactive mode
    read -p "Press Enter to start..."
else
    # Non-interactive mode
    print_info "Running in non-interactive mode (auto-starting in 2 seconds)..."
    sleep 2
fi

# Change to repo root for running commands
cd "$REPO_ROOT"

# Helper function to run a test with skip option
run_test() {
    local test_num=$1
    local test_name=$2
    local action_info=$3
    shift 3
    local cmd_args=("$@")
    
    # Skip if we're running a specific test and this isn't it
    if [ -n "$TEST_NUM" ] && [ "$TEST_NUM" != "$test_num" ]; then
        return 0
    fi
    
    print_header "Test $test_num: $test_name"
    print_info "Action needed: $action_info"
    echo ""
    
    if [ -t 0 ]; then
        read -p "Press Enter to run this test (or 's' to skip): " response
        if [ "$response" = "s" ] || [ "$response" = "S" ]; then
            print_warning "Skipping test $test_num"
            return 0
        fi
    fi
    
    go run ./cmd/plz-confirm "${cmd_args[@]}"
    
    if [ $? -ne 0 ]; then
        print_error "$test_name command failed"
        exit 1
    fi
    
    echo ""
    if [ -t 0 ] && [ -z "$TEST_NUM" ]; then
        read -p "Press Enter to continue to next test..."
    else
        sleep 1
    fi
}

# 1. CONFIRM
run_test "1" "Confirm Widget" "Click 'APPROVE' in the browser dialog" \
    confirm \
    --title "System Update Required" \
    --message "A critical security patch (v2.4.0) is available. Install now?" \
    --approve-text "Install & Restart" \
    --reject-text "Remind Me Later" \
    --wait-timeout 120 \
    --output table

# 2. SELECT
run_test "2" "Select Widget" "Select 'us-west-2' and click Confirm" \
    select \
    --title "Select Region" \
    --option us-east-1 \
    --option us-west-2 \
    --option eu-central-1 \
    --option ap-northeast-1 \
    --searchable \
    --wait-timeout 120 \
    --output table

# 3. FORM
run_test "3" "Form Widget" "Fill in the form (username, email required) and click Submit" \
    form \
    --title "Administrator Details" \
    --schema "$SCRIPT_DIR/test-form-schema.json" \
    --wait-timeout 120 \
    --output table

# 4. TABLE
run_test "4" "Table Widget" "Select 'server-2' (single row) and click Confirm" \
    table \
    --title "Select Server" \
    --data "$SCRIPT_DIR/test-table-data.json" \
    --columns name,status,region,cpu \
    --searchable \
    --wait-timeout 120 \
    --output table

# 5. UPLOAD
run_test "5" "Upload Widget" "Upload a file (or click Cancel if no file available) and click Confirm" \
    upload \
    --title "Upload Log Files" \
    --accept .log \
    --accept .txt \
    --accept text/plain \
    --multiple \
    --max-size 5242880 \
    --wait-timeout 120 \
    --output table

print_header "All Tests Completed!"
print_info "All widget commands executed successfully."

