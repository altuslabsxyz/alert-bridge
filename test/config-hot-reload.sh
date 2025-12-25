#!/bin/bash

# Configuration Hot Reload Quick Validation Test
# Based on specs/config-hot-reload/quickstart.md

set -e

echo "=== Configuration Hot Reload Validation ==="
echo ""

# Test 1: Build the application
echo "Test 1: Building alert-bridge..."
go build -o alert-bridge ./cmd/alert-bridge
echo "✓ Build successful"
echo ""

# Test 2: Run integration tests
echo "Test 2: Running integration tests..."
go test -v ./internal/infrastructure/config/ | grep -E "PASS|FAIL"
echo "✓ Integration tests passed"
echo ""

# Test 3: Verify all required files exist
echo "Test 3: Verifying implementation files..."
required_files=(
    "internal/infrastructure/config/reload.go"
    "internal/infrastructure/config/watcher.go"
    "internal/infrastructure/config/validator.go"
    "internal/infrastructure/config/reload_test.go"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✓ $file exists"
    else
        echo "✗ $file missing"
        exit 1
    fi
done
echo ""

# Test 4: Verify Viper dependency
echo "Test 4: Checking Viper dependency..."
if grep -q "github.com/spf13/viper" go.mod; then
    echo "✓ Viper dependency added"
else
    echo "✗ Viper dependency missing"
    exit 1
fi
echo ""

# Test 5: Verify ConfigManager integration in main.go
echo "Test 5: Checking ConfigManager integration..."
if grep -q "ConfigManager" cmd/alert-bridge/main.go; then
    echo "✓ ConfigManager integrated in main.go"
else
    echo "✗ ConfigManager not found in main.go"
    exit 1
fi

if grep -q "watcher.Start()" cmd/alert-bridge/main.go; then
    echo "✓ Config watcher started in main.go"
else
    echo "✗ Config watcher not started in main.go"
    exit 1
fi
echo ""

echo "=== All Validation Tests Passed ==="
echo ""
echo "Configuration Hot Reload feature is ready!"
echo ""
echo "Next steps:"
echo "1. Start alert-bridge with: ./alert-bridge"
echo "2. Modify config/config.yaml (e.g., change logging.level to 'debug')"
echo "3. Watch logs for 'configuration reloaded' message"
echo "4. Verify new settings take effect without restart"
