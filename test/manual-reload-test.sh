#!/bin/bash

# Manual Configuration Reload Test
# Tests POST /-/reload endpoint

set -e

echo "=== Manual Configuration Reload Test ==="
echo ""

# Check if alert-bridge is running
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "Error: alert-bridge is not running on localhost:8080"
    echo "Start it first: ./alert-bridge"
    exit 1
fi

echo "✓ alert-bridge is running"
echo ""

# Test 1: Reload current configuration
echo "Test 1: Reload current configuration"
response=$(curl -s -X POST http://localhost:8080/-/reload)
echo "Response: $response"

if echo "$response" | grep -q "successfully"; then
    echo "✓ Reload successful"
else
    echo "✗ Reload failed"
    exit 1
fi
echo ""

# Test 2: Check wrong method (GET should fail)
echo "Test 2: Test GET request (should fail with 405)"
status_code=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/-/reload)

if [ "$status_code" == "405" ]; then
    echo "✓ GET request correctly rejected with 405"
else
    echo "✗ Expected 405, got $status_code"
    exit 1
fi
echo ""

echo "=== Manual Reload Tests Passed ==="
echo ""
echo "To test logging format change:"
echo "1. Edit config/config.yaml:"
echo "   logging:"
echo "     level: debug"
echo "     format: text"
echo ""
echo "2. Reload configuration:"
echo "   curl -X POST http://localhost:8080/-/reload"
echo ""
echo "3. Check logs - should now be in text format with debug level"
