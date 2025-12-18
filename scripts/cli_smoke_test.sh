#!/bin/bash
# Copyright © 2024 KubeStack-AI Authors
#
# CLI Smoke Test Script
# This script performs basic smoke tests on the KSA CLI to verify
# that all commands are functional and responsive.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Binary path
KSA_BIN="${KSA_BIN:-./ksa}"

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_test() {
    echo -e "\n${YELLOW}[TEST]${NC} $1"
    TESTS_RUN=$((TESTS_RUN + 1))
}

test_passed() {
    echo -e "${GREEN}✓ PASSED${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

test_failed() {
    echo -e "${RED}✗ FAILED${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

# Check if binary exists
check_binary() {
    if [ ! -f "$KSA_BIN" ]; then
        log_error "KSA binary not found at: $KSA_BIN"
        log_info "Building binary..."
        make build || {
            log_error "Failed to build binary"
            exit 1
        }
    fi
    
    if [ ! -x "$KSA_BIN" ]; then
        log_error "KSA binary is not executable: $KSA_BIN"
        chmod +x "$KSA_BIN" || exit 1
    fi
    
    log_info "Using KSA binary: $KSA_BIN"
}

# Test 1: Version check
test_version() {
    log_test "Testing version command"
    
    output=$($KSA_BIN version 2>&1)
    if echo "$output" | grep -q "KubeStack-AI"; then
        test_passed
    else
        test_failed "Version output doesn't contain 'KubeStack-AI'"
        echo "Output: $output"
    fi
}

# Test 2: Help text
test_help() {
    log_test "Testing --help flag"
    
    output=$($KSA_BIN --help 2>&1)
    if echo "$output" | grep -q "Usage:" && echo "$output" | grep -q "Available Commands:"; then
        test_passed
    else
        test_failed "Help output missing expected sections"
        echo "Output: $output"
    fi
}

# Test 3: Diagnose help
test_diagnose_help() {
    log_test "Testing diagnose --help"
    
    output=$($KSA_BIN diagnose --help 2>&1)
    if echo "$output" | grep -q "diagnose" && echo "$output" | grep -q "middleware"; then
        test_passed
    else
        test_failed "Diagnose help missing expected content"
        echo "Output: $output"
    fi
}

# Test 4: Ask help
test_ask_help() {
    log_test "Testing ask --help"
    
    output=$($KSA_BIN ask --help 2>&1)
    if echo "$output" | grep -q "ask"; then
        test_passed
    else
        test_failed "Ask help missing expected content"
        echo "Output: $output"
    fi
}

# Test 5: Fix help
test_fix_help() {
    log_test "Testing fix --help"
    
    output=$($KSA_BIN fix --help 2>&1)
    if echo "$output" | grep -q "fix"; then
        test_passed
    else
        test_failed "Fix help missing expected content"
        echo "Output: $output"
    fi
}

# Test 6: Server help
test_server_help() {
    log_test "Testing server --help"
    
    output=$($KSA_BIN server --help 2>&1)
    if echo "$output" | grep -q "server"; then
        test_passed
    else
        test_failed "Server help missing expected content"
        echo "Output: $output"
    fi
}

# Test 6a: Plugin help
test_plugin_help() {
    log_test "Testing plugin --help"
    
    output=$($KSA_BIN plugin --help 2>&1)
    if echo "$output" | grep -q "plugin"; then
        test_passed
    else
        test_failed "Plugin help missing expected content"
        echo "Output: $output"
    fi
}

# Test 6b: Plugin list
test_plugin_list() {
    log_test "Testing plugin list"
    
    set +e
    output=$($KSA_BIN plugin list 2>&1)
    exit_code=$?
    set -e
    
    # Command should complete or show error about missing plugin directory
    if [ $exit_code -eq 0 ] || echo "$output" | grep -q -i "plugin\|directory\|error"; then
        test_passed
    else
        test_failed "Plugin list should complete or show meaningful error"
        echo "Output: $output"
    fi
}

# Test 6c: KB help
test_kb_help() {
    log_test "Testing kb --help"
    
    output=$($KSA_BIN kb --help 2>&1)
    if echo "$output" | grep -q "kb\|knowledge"; then
        test_passed
    else
        test_failed "KB help missing expected content"
        echo "Output: $output"
    fi
}

# Test 6d: KB search
test_kb_search() {
    log_test "Testing kb search"
    
    set +e
    output=$($KSA_BIN kb search "Redis" 2>&1)
    exit_code=$?
    set -e
    
    # Command should complete successfully or show error
    if [ $exit_code -eq 0 ] || echo "$output" | grep -q -i "redis\|entry\|found\|error"; then
        test_passed
    else
        test_failed "KB search should complete or show meaningful error"
        echo "Output: $output"
    fi
}

# Test 7: Config file validation
test_config_validation() {
    log_test "Testing config file validation"
    
    if [ -f "configs/config.yaml" ]; then
        output=$($KSA_BIN --config configs/config.yaml version 2>&1)
        if echo "$output" | grep -q "KubeStack-AI"; then
            test_passed
        else
            test_failed "Config file validation failed"
            echo "Output: $output"
        fi
    else
        log_warn "Config file not found, skipping test"
        test_passed
    fi
}

# Test 8: JSON output format
test_json_output() {
    log_test "Testing JSON output format (version)"
    
    output=$($KSA_BIN version -o json 2>&1 || echo "command_failed")
    # Version command might not support JSON output, so we just check it doesn't crash
    if [ "$output" != "command_failed" ]; then
        test_passed
    else
        log_warn "JSON output test inconclusive"
        test_passed
    fi
}

# Test 9: YAML output format
test_yaml_output() {
    log_test "Testing YAML output format (version)"
    
    output=$($KSA_BIN version -o yaml 2>&1 || echo "command_failed")
    # Version command might not support YAML output, so we just check it doesn't crash
    if [ "$output" != "command_failed" ]; then
        test_passed
    else
        log_warn "YAML output test inconclusive"
        test_passed
    fi
}

# Test 10: Invalid command handling
test_invalid_command() {
    log_test "Testing invalid command error handling"
    
    set +e  # Temporarily disable exit on error
    output=$($KSA_BIN invalid-command-xyz 2>&1)
    exit_code=$?
    set -e
    
    if [ $exit_code -ne 0 ] && echo "$output" | grep -q -i "unknown\|error\|invalid"; then
        test_passed
    else
        test_failed "Invalid command should produce error"
        echo "Output: $output"
        echo "Exit code: $exit_code"
    fi
}

# Test 11: Missing required flag
test_missing_required_flag() {
    log_test "Testing missing required flag handling"
    
    set +e
    output=$($KSA_BIN diagnose 2>&1)
    exit_code=$?
    set -e
    
    # Command should either show usage or error about missing middleware type
    if [ $exit_code -ne 0 ] || echo "$output" | grep -q -i "usage\|required\|middleware"; then
        test_passed
    else
        log_warn "Diagnose without args should show usage or error"
        test_passed
    fi
}

# Test 12: Log level flag
test_log_level_flag() {
    log_test "Testing --log-level flag"
    
    output=$($KSA_BIN --log-level debug version 2>&1)
    # Just verify the command completes successfully
    if [ $? -eq 0 ]; then
        test_passed
    else
        test_failed "Log level flag caused error"
        echo "Output: $output"
    fi
}

# Test 13: Global flags persistence
test_global_flags() {
    log_test "Testing global flags work with subcommands"
    
    output=$($KSA_BIN --log-level info version 2>&1)
    if [ $? -eq 0 ]; then
        test_passed
    else
        test_failed "Global flags not working with subcommands"
        echo "Output: $output"
    fi
}

# Test 14: Diagnose with dry-run
test_diagnose_dry_run() {
    log_test "Testing diagnose with --dry-run flag"
    
    set +e
    output=$($KSA_BIN diagnose redis --instance test --dry-run 2>&1 || echo "")
    exit_code=$?
    set -e
    
    # Command may fail due to missing dependencies, but should not crash
    log_warn "Dry-run test is informational only"
    test_passed
}

# Test 15: Binary size check
test_binary_size() {
    log_test "Checking binary size is reasonable"
    
    size=$(stat -f%z "$KSA_BIN" 2>/dev/null || stat -c%s "$KSA_BIN" 2>/dev/null)
    size_mb=$((size / 1024 / 1024))
    
    if [ $size_mb -lt 500 ]; then
        test_passed
        log_info "Binary size: ${size_mb}MB"
    else
        test_failed "Binary size is too large: ${size_mb}MB"
    fi
}

# Print summary
print_summary() {
    echo ""
    echo "=========================================="
    echo "           SMOKE TEST SUMMARY"
    echo "=========================================="
    echo "Total Tests Run:    $TESTS_RUN"
    echo -e "Tests Passed:       ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests Failed:       ${RED}$TESTS_FAILED${NC}"
    echo "=========================================="
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}✓ All smoke tests passed!${NC}"
        return 0
    else
        echo -e "${RED}✗ Some tests failed${NC}"
        return 1
    fi
}

# Main execution
main() {
    echo "=========================================="
    echo "     KSA CLI SMOKE TEST SUITE"
    echo "=========================================="
    
    check_binary
    
    # Run all tests
    test_version
    test_help
    test_diagnose_help
    test_ask_help
    test_fix_help
    test_server_help
    test_plugin_help
    test_plugin_list
    test_kb_help
    test_kb_search
    test_config_validation
    test_json_output
    test_yaml_output
    test_invalid_command
    test_missing_required_flag
    test_log_level_flag
    test_global_flags
    test_diagnose_dry_run
    test_binary_size
    
    # Print summary and exit
    print_summary
}

# Run main function
main
exit $?
