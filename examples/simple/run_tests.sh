#!/bin/bash
# Automated test runner for ConsoleKit
# Tests the stdin piping feature for CI/CD integration

set -e

APP="./simple"
if [ ! -f "$APP" ]; then
    APP="../../build/simple"
fi

if [ ! -f "$APP" ]; then
    echo "Error: simple app not found. Run 'make simple' first."
    exit 1
fi

echo "============================================"
echo "ConsoleKit Automated Test Suite"
echo "============================================"
echo

# Test 1: Basic piping
echo "Test 1: Basic script piping"
cat test.run | $APP > /tmp/test1.out 2>&1
if grep -q "AAA" /tmp/test1.out; then
    echo "✓ Test 1 PASSED"
else
    echo "✗ Test 1 FAILED"
    cat /tmp/test1.out
    exit 1
fi
echo

# Test 2: Error handling (should continue despite errors)
echo "Test 2: Error handling"
cat test_fail.run | $APP > /tmp/test2.out 2>&1
if grep -q "test1 . run script" /tmp/test2.out && grep -q "Error at line 2" /tmp/test2.out; then
    echo "✓ Test 2 PASSED (errors handled correctly)"
else
    echo "✗ Test 2 FAILED"
    cat /tmp/test2.out
    exit 1
fi
echo

# Test 3: Here document
echo "Test 3: Here document"
$APP << 'EOF' > /tmp/test3.out 2>&1
print "Dynamic test"
print "Multi-line execution"
EOF

if grep -q "Dynamic test" /tmp/test3.out && grep -q "Multi-line execution" /tmp/test3.out; then
    echo "✓ Test 3 PASSED"
else
    echo "✗ Test 3 FAILED"
    cat /tmp/test3.out
    exit 1
fi
echo

# Test 4: Comments and empty lines
echo "Test 4: Comments and empty lines"
{
  echo "# This is a comment"
  echo ""
  echo "print test"
  echo ""
  echo "# Another comment"
  echo "print done"
} | $APP > /tmp/test4.out 2>&1

if grep -q "test" /tmp/test4.out && grep -q "done" /tmp/test4.out; then
    echo "✓ Test 4 PASSED"
else
    echo "✗ Test 4 FAILED"
    cat /tmp/test4.out
    exit 1
fi
echo

# Test 5: Single command piping
echo "Test 5: Single command piping"
echo "print Hello World" | $APP > /tmp/test5.out 2>&1
if grep -q "Hello World" /tmp/test5.out; then
    echo "✓ Test 5 PASSED"
else
    echo "✗ Test 5 FAILED"
    cat /tmp/test5.out
    exit 1
fi
echo

# Test 6: Comprehensive test script
echo "Test 6: Comprehensive test script"
if [ -f "test_comprehensive.run" ]; then
    cat test_comprehensive.run | $APP > /tmp/test6.out 2>&1
    if grep -q "All tests completed successfully" /tmp/test6.out; then
        echo "✓ Test 6 PASSED"
    else
        echo "✗ Test 6 FAILED"
        cat /tmp/test6.out
        exit 1
    fi
else
    echo "⊘ Test 6 SKIPPED (test_comprehensive.run not found)"
fi
echo

echo "============================================"
echo "All Tests PASSED ✓"
echo "============================================"
echo
echo "ConsoleKit stdin piping is working correctly!"
echo "You can now use it for automated testing:"
echo "  cat your_script.run | $APP"
echo
