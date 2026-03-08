#!/bin/bash

###############################################################################
# AI Knowledge Cache Refresh Script
#
# Description:
#   This script refreshes AI knowledge base cache from database to Redis
#   by calling the refresh API endpoint.
#
# Crontab Example (store token in environment variable or file):
#   # Set token in crontab environment
#   AUTH_TOKEN=eyJhbGciOiJIUzI1NiIs...
#
#   # Refresh AI knowledge cache every day at 2 AM
#   0 2 * * * /path/to/il-dashboard/scripts/refresh_ai_knowledge.sh https://ilbedev.kitamandiri.com $AUTH_TOKEN >> /var/log/ai-knowledge-refresh.log 2>&1
#
#   # Or load token from file
#   0 2 * * * /path/to/il-dashboard/scripts/refresh_ai_knowledge.sh https://ilbedev.kitamandiri.com $(cat /path/to/token.txt) >> /var/log/ai-knowledge-refresh.log 2>&1
#
# Usage:
#   ./refresh_ai_knowledge.sh <base_url> <auth_token>
#
# Example:
#   ./refresh_ai_knowledge.sh https://ilbedev.kitamandiri.com eyJhbGciOiJIUzI1NiIs...
#
###############################################################################

# Configuration
BASE_URL="${1:-https://ilbedev.kitamandiri.com}"
AUTH_TOKEN="${2:-}"  # Pass token as second argument
API_ENDPOINT="${BASE_URL}/api/v1/instant-link/ai/knowledge/refresh-all"
TIMEOUT=30

# Check if AUTH_TOKEN is provided
if [ -z "$AUTH_TOKEN" ]; then
  echo "ERROR: Authentication token is required!"
  echo "Usage: $0 <base_url> <auth_token>"
  echo "Example: $0 https://ilbedev.kitamandiri.com eyJhbGciOiJIUzI1NiIs..."
  exit 1
fi

# Logging
LOG_DATE=$(date '+%Y-%m-%d %H:%M:%S')

echo "========================================="
echo "[$LOG_DATE] Starting AI Knowledge Cache Refresh"
echo "========================================="
echo "API Endpoint: $API_ENDPOINT"
echo ""

# Make API call
HTTP_CODE=$(curl -s -o /tmp/ai_refresh_response.txt -w "%{http_code}" \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  --max-time $TIMEOUT \
  "$API_ENDPOINT")

# Check response
if [ "$HTTP_CODE" -eq 200 ]; then
  echo "✓ SUCCESS: Knowledge cache refreshed successfully"
  echo "Response:"
  cat /tmp/ai_refresh_response.txt | jq '.' 2>/dev/null || cat /tmp/ai_refresh_response.txt
  EXIT_CODE=0
else
  echo "✗ ERROR: Failed to refresh knowledge cache"
  echo "HTTP Status Code: $HTTP_CODE"
  echo "Response:"
  cat /tmp/ai_refresh_response.txt
  EXIT_CODE=1
fi

# Cleanup
rm -f /tmp/ai_refresh_response.txt

echo ""
echo "[$LOG_DATE] AI Knowledge Cache Refresh completed with exit code: $EXIT_CODE"
echo "========================================="
echo ""

exit $EXIT_CODE
