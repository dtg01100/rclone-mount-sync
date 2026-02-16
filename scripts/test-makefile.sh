#!/bin/bash
#
# Test script for Makefile install/uninstall targets
# Tests the install and uninstall functionality without requiring root privileges
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Project root directory (relative to script location)
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY_NAME="rclone-mount-sync"
BUILD_DIR="bin"

# Temporary directory for testing
TEMP_DIR=""

# Test installation prefix (used with make PREFIX=...)
TEST_PREFIX=""

# Cleanup function
cleanup() {
    if [[ -n "$TEMP_DIR" && -d "$TEMP_DIR" ]]; then
        rm -rf "$TEMP_DIR"
    fi
    # Clean up build artifacts
    cd "$PROJECT_ROOT" && make clean >/dev/null 2>&1 || true
}

# Set up cleanup on exit
trap cleanup EXIT

# Print test result
pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    if [[ -n "$2" ]]; then
        echo -e "  ${YELLOW}Details${NC}: $2"
    fi
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

# Set up temporary test environment
setup_temp_dir() {
    TEMP_DIR=$(mktemp -d)
    TEST_PREFIX="$TEMP_DIR/usr/local"
    echo -e "${YELLOW}Setting up test environment${NC}: $TEMP_DIR"
    echo -e "${YELLOW}Test PREFIX${NC}: $TEST_PREFIX"
}

# ==============================================================================
# TEST: Build succeeds before install
# ==============================================================================
test_build_succeeds() {
    echo ""
    echo "=== Test: Build succeeds before install ==="
    
    cd "$PROJECT_ROOT"
    
    if make build >/dev/null 2>&1; then
        if [[ -f "$BUILD_DIR/$BINARY_NAME" ]]; then
            pass "make build succeeds and creates binary"
        else
            fail "make build succeeded but binary not found" "Expected: $BUILD_DIR/$BINARY_NAME"
        fi
    else
        fail "make build failed" "Check build output for errors"
    fi
}

# ==============================================================================
# TEST: Binary is copied to destination with correct permissions
# ==============================================================================
test_install_copies_binary() {
    echo ""
    echo "=== Test: Install copies binary to destination ==="
    
    local DEST_DIR="$TEMP_DIR/usr/local/bin"
    local DEST_BINARY="$DEST_DIR/$BINARY_NAME"
    
    cd "$PROJECT_ROOT"
    
    # First ensure binary exists
    if ! make build >/dev/null 2>&1; then
        fail "Build failed, cannot test install" "Run 'make build' manually to check errors"
        return
    fi
    
    # Run make install with PREFIX pointing to temp directory
    if make install PREFIX="$TEST_PREFIX" >/dev/null 2>&1; then
        if [[ -f "$DEST_BINARY" ]]; then
            pass "Binary copied to destination"
        else
            fail "Binary not found at destination" "Expected: $DEST_BINARY"
        fi
    else
        fail "make install failed" "Could not install binary to $DEST_DIR"
    fi
}

# ==============================================================================
# TEST: Binary has correct permissions (755)
# ==============================================================================
test_install_permissions() {
    echo ""
    echo "=== Test: Binary has correct permissions (755) ==="
    
    local DEST_DIR="$TEMP_DIR/usr/local/bin"
    local DEST_BINARY="$DEST_DIR/$BINARY_NAME"
    
    cd "$PROJECT_ROOT"
    
    # Ensure binary exists
    if ! make build >/dev/null 2>&1; then
        fail "Build failed" "Cannot test permissions"
        return
    fi
    
    # Run make install with PREFIX pointing to temp directory
    make install PREFIX="$TEST_PREFIX" >/dev/null 2>&1
    
    # Check permissions
    local perms
    perms=$(stat -c "%a" "$DEST_BINARY" 2>/dev/null || stat -f "%Lp" "$DEST_BINARY" 2>/dev/null)
    
    if [[ "$perms" == "755" ]]; then
        pass "Binary has correct permissions (755)"
    else
        fail "Binary has incorrect permissions" "Expected: 755, Got: $perms"
    fi
}

# ==============================================================================
# TEST: Binary is executable
# ==============================================================================
test_install_executable() {
    echo ""
    echo "=== Test: Binary is executable ==="
    
    local DEST_DIR="$TEMP_DIR/usr/local/bin"
    local DEST_BINARY="$DEST_DIR/$BINARY_NAME"
    
    cd "$PROJECT_ROOT"
    
    # Ensure binary exists
    if ! make build >/dev/null 2>&1; then
        fail "Build failed" "Cannot test executable status"
        return
    fi
    
    # Run make install with PREFIX pointing to temp directory
    make install PREFIX="$TEST_PREFIX" >/dev/null 2>&1
    
    if [[ -x "$DEST_BINARY" ]]; then
        pass "Binary is executable"
    else
        fail "Binary is not executable" "File does not have execute permission"
    fi
}

# ==============================================================================
# TEST: Uninstall removes binary
# ==============================================================================
test_uninstall_removes_binary() {
    echo ""
    echo "=== Test: Uninstall removes binary ==="
    
    local DEST_DIR="$TEMP_DIR/usr/local/bin"
    local DEST_BINARY="$DEST_DIR/$BINARY_NAME"
    
    cd "$PROJECT_ROOT"
    
    if ! make build >/dev/null 2>&1; then
        fail "Build failed" "Cannot test uninstall"
        return
    fi
    
    # Install using make install with PREFIX
    make install PREFIX="$TEST_PREFIX" >/dev/null 2>&1
    
    # Verify binary exists
    if [[ ! -f "$DEST_BINARY" ]]; then
        fail "Setup failed" "Binary not installed for uninstall test"
        return
    fi
    
    # Run make uninstall with PREFIX
    if make uninstall PREFIX="$TEST_PREFIX" >/dev/null 2>&1; then
        if [[ ! -f "$DEST_BINARY" ]]; then
            pass "Uninstall removes binary"
        else
            fail "Uninstall did not remove binary" "File still exists: $DEST_BINARY"
        fi
    else
        fail "make uninstall failed" "Uninstall command returned error"
    fi
}

# ==============================================================================
# TEST: Uninstall is idempotent (running twice doesn't error)
# ==============================================================================
test_uninstall_idempotent() {
    echo ""
    echo "=== Test: Uninstall is idempotent ==="
    
    local DEST_DIR="$TEMP_DIR/usr/local/bin"
    local DEST_BINARY="$DEST_DIR/$BINARY_NAME"
    
    cd "$PROJECT_ROOT"
    
    if ! make build >/dev/null 2>&1; then
        fail "Build failed" "Cannot test idempotent uninstall"
        return
    fi
    
    # Install binary using make install
    make install PREFIX="$TEST_PREFIX" >/dev/null 2>&1
    
    # First uninstall using make uninstall
    make uninstall PREFIX="$TEST_PREFIX" >/dev/null 2>&1
    
    # Second uninstall (should not error)
    if make uninstall PREFIX="$TEST_PREFIX" >/dev/null 2>&1; then
        pass "Uninstall is idempotent (no error on second run)"
    else
        fail "Second uninstall failed" "make uninstall should not fail on non-existent file"
    fi
}

# ==============================================================================
# TEST: Install overwrites existing binary
# ==============================================================================
test_install_overwrites_existing() {
    echo ""
    echo "=== Test: Install overwrites existing binary ==="
    
    local DEST_DIR="$TEMP_DIR/usr/local/bin"
    local DEST_BINARY="$DEST_DIR/$BINARY_NAME"
    
    cd "$PROJECT_ROOT"
    
    if ! make build >/dev/null 2>&1; then
        fail "Build failed" "Cannot test overwrite"
        return
    fi
    
    # Create a dummy file at destination
    mkdir -p "$DEST_DIR"
    echo "dummy content" > "$DEST_BINARY"
    local old_inode
    old_inode=$(stat -c "%i" "$DEST_BINARY" 2>/dev/null || stat -f "%i" "$DEST_BINARY" 2>/dev/null)
    
    # Install over existing file using make install
    if make install PREFIX="$TEST_PREFIX" >/dev/null 2>&1; then
        # Check that file was replaced (different inode or content changed)
        local new_inode
        new_inode=$(stat -c "%i" "$DEST_BINARY" 2>/dev/null || stat -f "%i" "$DEST_BINARY" 2>/dev/null)
        
        # Check file is now a valid binary (not our dummy text)
        if file "$DEST_BINARY" 2>/dev/null | grep -q "executable\|ELF\|binary"; then
            pass "Install overwrites existing binary"
        elif [[ "$old_inode" != "$new_inode" ]]; then
            pass "Install overwrites existing binary (file replaced)"
        else
            # Check content changed
            if ! grep -q "dummy content" "$DEST_BINARY" 2>/dev/null; then
                pass "Install overwrites existing binary (content changed)"
            else
                fail "Install did not overwrite existing file" "File still contains dummy content"
            fi
        fi
    else
        fail "Install failed to overwrite existing file" "make install command returned error"
    fi
}

# ==============================================================================
# TEST: Uninstall succeeds even if binary doesn't exist
# ==============================================================================
test_uninstall_nonexistent() {
    echo ""
    echo "=== Test: Uninstall succeeds on non-existent binary ==="
    
    cd "$PROJECT_ROOT"
    
    # Use a unique PREFIX where binary definitely doesn't exist
    local NONEXISTENT_PREFIX="$TEMP_DIR/nonexistent"
    
    # Try to uninstall non-existent file using make uninstall
    if make uninstall PREFIX="$NONEXISTENT_PREFIX" >/dev/null 2>&1; then
        pass "Uninstall succeeds on non-existent binary"
    else
        fail "Uninstall failed on non-existent binary" "make uninstall should always succeed"
    fi
}

# ==============================================================================
# Main test runner
# ==============================================================================
main() {
    echo "========================================"
    echo "  Makefile Install/Uninstall Tests"
    echo "========================================"
    echo ""
    echo "Project root: $PROJECT_ROOT"
    
    # Setup
    setup_temp_dir
    
    # Run tests
    test_build_succeeds
    test_install_copies_binary
    test_install_permissions
    test_install_executable
    test_uninstall_removes_binary
    test_uninstall_idempotent
    test_install_overwrites_existing
    test_uninstall_nonexistent
    
    # Print summary
    echo ""
    echo "========================================"
    echo "  Test Summary"
    echo "========================================"
    echo -e "${GREEN}Passed${NC}: $TESTS_PASSED"
    echo -e "${RED}Failed${NC}: $TESTS_FAILED"
    echo ""
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    else
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    fi
}

# Run main
main "$@"
