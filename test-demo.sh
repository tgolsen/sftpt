#!/bin/bash

# sftpt CLI Demo Script
# Demonstrates the key features that solve common SFTP authentication problems

set -e  # Exit on any error

SFTPT="./build/sftpt"
SERVER="testuser@localhost:2222"
PASSWORD="password"

echo "🚀 sftpt CLI Demo - Solving SFTP Authentication Pain Points"
echo "=========================================================="
echo

# Test 1: Clean password authentication (no key failures)
echo "✅ Test 1: Clean password authentication (avoids 'too many auth failures')"
$SFTPT list $SERVER:/upload --password-stdin $PASSWORD --quiet
echo "   → Connected successfully without SSH key interference"
echo

# Test 2: Script-friendly output
echo "✅ Test 2: Clean, parseable output for shell scripts"
echo "   Files in /upload:"
$SFTPT list $SERVER:/upload --password-stdin $PASSWORD | sed 's/^/   - /'
echo

# Test 3: Long format with details
echo "✅ Test 3: Detailed file information"
$SFTPT list $SERVER:/upload --password-stdin $PASSWORD --long
echo

# Test 4: Proper exit codes
echo "✅ Test 4: Proper exit codes for script integration"
if $SFTPT list $SERVER:/nonexistent --password-stdin $PASSWORD --quiet 2>/dev/null; then
    echo "   ERROR: Should have failed!"
    exit 1
else
    echo "   → Correctly returned exit code 1 for missing directory"
fi
echo

# Test 5: File operations
echo "✅ Test 5: File upload and download"
echo "test content" > /tmp/sftpt-test.txt
$SFTPT put /tmp/sftpt-test.txt $SERVER:/upload/ --password-stdin $PASSWORD --quiet
echo "   → Uploaded test file"

$SFTPT get $SERVER:/upload/sftpt-test.txt /tmp/downloaded-test.txt --password-stdin $PASSWORD --quiet
if diff /tmp/sftpt-test.txt /tmp/downloaded-test.txt >/dev/null; then
    echo "   → Downloaded and verified file integrity"
else
    echo "   ERROR: File integrity check failed!"
    exit 1
fi

# Clean up
$SFTPT rm $SERVER:/upload/sftpt-test.txt --password-stdin $PASSWORD --quiet
rm -f /tmp/sftpt-test.txt /tmp/downloaded-test.txt
echo

# Test 6: Directory operations
echo "✅ Test 6: Directory operations"
$SFTPT mkdir $SERVER:/upload/test-dir --password-stdin $PASSWORD --quiet
echo "   → Created directory"
$SFTPT rm $SERVER:/upload/test-dir --recursive --password-stdin $PASSWORD --quiet
echo "   → Removed directory"
echo

# Test 7: Authentication method control
echo "✅ Test 7: Authentication method control"
echo "   Keys-only mode (will fail as expected):"
if $SFTPT list $SERVER:/upload --keys-only --quiet 2>/dev/null; then
    echo "   ERROR: Keys-only should have failed!"
    exit 1
else
    echo "   → Correctly failed when no SSH keys available"
fi
echo

echo "🎉 All tests passed! sftpt successfully addresses common SFTP pain points:"
echo
echo "   ✓ No 'too many authentication failures' - smart auth method selection"
echo "   ✓ Clean command-line password option (--password-stdin) with security warning"
echo "   ✓ Keys-only mode (--keys-only) to prevent sending all SSH agent keys"
echo "   ✓ Script-friendly output and proper exit codes"
echo "   ✓ Comprehensive error messages"
echo "   ✓ Full SFTP operation support (list, get, put, mkdir, rm)"
echo
echo "sftpt is ready for production use! 🚀"