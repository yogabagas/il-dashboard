#!/bin/bash

# JSON Logger Test Script
# This script tests the JSON logging implementation

echo "=========================================="
echo "JSON Logger Implementation Test"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Check if logger package exists
echo -n "Test 1: Checking logger package... "
if [ -d "logger" ]; then
    echo -e "${GREEN}✓ PASS${NC}"
else
    echo -e "${RED}✗ FAIL${NC}"
    exit 1
fi

# Test 2: Check if configuration exists
echo -n "Test 2: Checking configuration... "
if grep -q "app.log.json.enabled" config-dev.properties; then
    echo -e "${GREEN}✓ PASS${NC}"
else
    echo -e "${RED}✗ FAIL${NC}"
    exit 1
fi

# Test 3: Check if main.go imports custom logger
echo -n "Test 3: Checking main.go imports... "
if grep -q "customLogger.*logger" main.go; then
    echo -e "${GREEN}✓ PASS${NC}"
else
    echo -e "${RED}✗ FAIL${NC}"
    exit 1
fi

# Test 4: Build the application
echo -n "Test 4: Building application... "
if go build -o /tmp/il-dashboard-test 2>/dev/null; then
    echo -e "${GREEN}✓ PASS${NC}"
    rm -f /tmp/il-dashboard-test
else
    echo -e "${RED}✗ FAIL${NC}"
    echo ""
    echo "Build errors:"
    go build 2>&1 | head -20
    exit 1
fi

# Test 5: Check if log directory can be created
echo -n "Test 5: Creating log directory... "
mkdir -p logs/il-dashboard 2>/dev/null
if [ -d "logs/il-dashboard" ]; then
    echo -e "${GREEN}✓ PASS${NC}"
else
    echo -e "${RED}✗ FAIL${NC}"
    exit 1
fi

# Test 6: Check go.mod for dependencies
echo -n "Test 6: Checking dependencies... "
if grep -q "github.com/sirupsen/logrus" go.mod; then
    echo -e "${GREEN}✓ PASS${NC}"
else
    echo -e "${RED}✗ FAIL${NC}"
    exit 1
fi

echo ""
echo "=========================================="
echo -e "${GREEN}All tests passed!${NC}"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. Run: go build"
echo "2. Run: ./il-dashboard"
echo "3. Check: logs/il-dashboard/app-$(date +%Y-%m-%d).json"
echo "4. View: tail -f logs/il-dashboard/app-*.json | jq '.'"
echo ""
echo "Documentation:"
echo "- Quick Start: logger/QUICK_START.md"
echo "- Full Guide: logger/README.md"
echo "- Examples: logger/EXAMPLE_OUTPUT.md"
echo "- Summary: JSON_LOGGING_IMPLEMENTATION.md"
echo ""
