# Dashboard API Documentation

Base URL: `/api/v1/instant-link/dashboard`

**Authorization Required:** All endpoints require authentication with role `ADMIN` or `OWNER`

**Important Notes:**
- **ADMIN** users can view dashboard overview for all clients or filter by specific client
- **OWNER** users can only view dashboard overview for their own client (based on `authInfo.ClientId`)
- Dashboard provides message statistics for three time periods: today, last 7 days, and last 30 days

---

## 1. Get Dashboard Overview

**Endpoint:** `GET /api/v1/instant-link/dashboard/overview`

**Authorization:** ADMIN, OWNER

**Description:** Retrieve comprehensive dashboard statistics showing message delivery metrics for multiple time periods. Includes total messages sent, delivered, failed, and success rates.

**Headers:**
```
Authorization: Bearer <your-token>
```

**Query Parameters:**
- `clientId` (optional): Filter by specific client ID (ADMIN only)
  - If provided by ADMIN: shows data for that specific client
  - If not provided by ADMIN: shows data for ALL clients (aggregated)
  - If provided by OWNER: ignored (always shows own client data)

**cURL Example - ADMIN viewing all clients:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/dashboard/overview" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

**cURL Example - ADMIN viewing specific client:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/dashboard/overview?clientId=01HZQK5X8V9Y2N1P3R4T6W8Z0A" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

**cURL Example - OWNER viewing own client:**
```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/dashboard/overview" \
  -H "Authorization: Bearer YOUR_OWNER_TOKEN"
```

**Success Response (200 OK):**
```json
{
  "code": 0,
  "messages": "Success",
  "data": {
    "today": {
      "totalSent": 1250,
      "totalDelivered": 1180,
      "totalFailed": 70,
      "successRate": 94.4
    },
    "last7Days": {
      "totalSent": 8500,
      "totalDelivered": 8100,
      "totalFailed": 400,
      "successRate": 95.29
    },
    "last30Days": {
      "totalSent": 35000,
      "totalDelivered": 33250,
      "totalFailed": 1750,
      "successRate": 95.0
    }
  }
}
```

**Response Fields:**

**Root Level:**
- `code`: Status code (0 = success)
- `messages`: Status message
- `data`: Dashboard overview object containing three time period statistics

**Period Statistics (today, last7Days, last30Days):**
- `totalSent`: Total number of outbound messages sent in this period
- `totalDelivered`: Total number of messages successfully delivered (includes "delivered" and "read" status)
- `totalFailed`: Total number of messages that failed to send
- `successRate`: Percentage of delivered messages (0-100)

**Time Period Definitions:**
- **today**: Statistics from 00:00:00 today to current time
- **last7Days**: Statistics from 7 days ago (00:00:00) to current time
- **last30Days**: Statistics from 30 days ago (00:00:00) to current time

**Error Responses:**

500 Internal Server Error - Database query failed:
```json
{
  "code": 500,
  "message": "Database error occurred"
}
```

401 Unauthorized - Missing or invalid token:
```json
{
  "code": 401,
  "message": "Unauthorized"
}
```

403 Forbidden - User doesn't have required role:
```json
{
  "code": 403,
  "message": "Forbidden"
}
```

---

## Dashboard Statistics Explanation

### Message Counting Logic

**Total Sent (`totalSent`):**
- Counts ALL outbound messages in the time period
- Includes: sent, delivered, read, and failed status
- Query: `direction = 'outbound' AND created_at >= start_time AND created_at < end_time`

**Total Delivered (`totalDelivered`):**
- Counts only successfully delivered messages
- Includes messages with status: `delivered` OR `read`
- Query: `direction = 'outbound' AND status IN ('delivered', 'read') AND created_at >= start_time AND created_at < end_time`

**Total Failed (`totalFailed`):**
- Counts only failed messages
- Includes messages with status: `failed`
- Query: `direction = 'outbound' AND status = 'failed' AND created_at >= start_time AND created_at < end_time`

**Success Rate (`successRate`):**
- Calculated as: `(totalDelivered / totalSent) × 100`
- Returns 0 if totalSent = 0
- Represents delivery success percentage

### Example Calculation:

If you have:
- 1000 messages sent
- 950 delivered
- 50 failed

Then:
- `totalSent`: 1000
- `totalDelivered`: 950
- `totalFailed`: 50
- `successRate`: (950 / 1000) × 100 = 95.0%

---

## Common Use Cases

### 1. Monitor Today's Performance

View real-time statistics for messages sent today:

```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/dashboard/overview" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Then extract the `today` object from response.

---

### 2. Weekly Performance Review

Check performance for the last 7 days:

```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/dashboard/overview" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Then extract the `last7Days` object from response.

---

### 3. Monthly Analytics (ADMIN)

View aggregated statistics across all clients for the past 30 days:

```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/dashboard/overview" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

The response will show combined data from all clients.

---

### 4. Client-Specific Dashboard (ADMIN)

View dashboard for a specific client:

```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/dashboard/overview?clientId=01HZQK5X8V9Y2N1P3R4T6W8Z0A" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

---

### 5. Compare Time Periods

Fetch all periods and compare:

```bash
curl -X GET "http://localhost:8080/api/v1/instant-link/dashboard/overview" \
  -H "Authorization: Bearer YOUR_TOKEN" | jq '{
    today: .data.today.successRate,
    last7Days: .data.last7Days.successRate,
    last30Days: .data.last30Days.successRate
  }'
```

This shows success rate trends over time.

---

## Integration Examples

### Python Example - Dashboard Monitoring

```python
import requests

url = "http://localhost:8080/api/v1/instant-link/dashboard/overview"
headers = {
    "Authorization": "Bearer YOUR_TOKEN_HERE"
}

response = requests.get(url, headers=headers)
data = response.json()

if data["code"] == 0:
    dashboard = data["data"]

    # Print Today's Stats
    print("=== TODAY'S PERFORMANCE ===")
    print(f"Total Sent: {dashboard['today']['totalSent']}")
    print(f"Total Delivered: {dashboard['today']['totalDelivered']}")
    print(f"Total Failed: {dashboard['today']['totalFailed']}")
    print(f"Success Rate: {dashboard['today']['successRate']:.2f}%")

    # Print Last 7 Days Stats
    print("\n=== LAST 7 DAYS PERFORMANCE ===")
    print(f"Total Sent: {dashboard['last7Days']['totalSent']}")
    print(f"Total Delivered: {dashboard['last7Days']['totalDelivered']}")
    print(f"Total Failed: {dashboard['last7Days']['totalFailed']}")
    print(f"Success Rate: {dashboard['last7Days']['successRate']:.2f}%")

    # Alert if success rate drops below 90%
    if dashboard['today']['successRate'] < 90:
        print("\n⚠️  WARNING: Today's success rate is below 90%!")
else:
    print(f"Error: {data['message']}")
```

---

### JavaScript Example - Dashboard Widget

```javascript
const axios = require('axios');

async function getDashboard(clientId = null) {
  const url = 'http://localhost:8080/api/v1/instant-link/dashboard/overview';
  const params = clientId ? { clientId } : {};

  const response = await axios.get(url, {
    headers: {
      'Authorization': `Bearer ${process.env.API_TOKEN}`
    },
    params: params
  });

  if (response.data.code === 0) {
    const dashboard = response.data.data;

    return {
      today: {
        sent: dashboard.today.totalSent,
        delivered: dashboard.today.totalDelivered,
        failed: dashboard.today.totalFailed,
        successRate: dashboard.today.successRate.toFixed(2) + '%'
      },
      weekly: {
        sent: dashboard.last7Days.totalSent,
        delivered: dashboard.last7Days.totalDelivered,
        failed: dashboard.last7Days.totalFailed,
        successRate: dashboard.last7Days.successRate.toFixed(2) + '%'
      },
      monthly: {
        sent: dashboard.last30Days.totalSent,
        delivered: dashboard.last30Days.totalDelivered,
        failed: dashboard.last30Days.totalFailed,
        successRate: dashboard.last30Days.successRate.toFixed(2) + '%'
      }
    };
  }
}

// Usage - All clients (ADMIN)
getDashboard().then(data => console.log('All Clients:', data));

// Usage - Specific client (ADMIN)
getDashboard('01HZQK5X8V9Y2N1P3R4T6W8Z0A').then(data =>
  console.log('Client Dashboard:', data)
);
```

---

### Bash Script Example - Daily Status Report

```bash
#!/bin/bash

TOKEN="YOUR_TOKEN_HERE"
URL="http://localhost:8080/api/v1/instant-link/dashboard/overview"

echo "========================================="
echo "     DAILY MESSAGING STATUS REPORT"
echo "========================================="
echo ""

# Fetch dashboard data
RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" "$URL")

# Extract today's stats using jq
TODAY_SENT=$(echo $RESPONSE | jq -r '.data.today.totalSent')
TODAY_DELIVERED=$(echo $RESPONSE | jq -r '.data.today.totalDelivered')
TODAY_FAILED=$(echo $RESPONSE | jq -r '.data.today.totalFailed')
TODAY_RATE=$(echo $RESPONSE | jq -r '.data.today.successRate')

echo "📊 TODAY'S STATISTICS:"
echo "   Messages Sent:      $TODAY_SENT"
echo "   Messages Delivered: $TODAY_DELIVERED"
echo "   Messages Failed:    $TODAY_FAILED"
echo "   Success Rate:       $TODAY_RATE%"
echo ""

# Extract last 7 days stats
WEEK_SENT=$(echo $RESPONSE | jq -r '.data.last7Days.totalSent')
WEEK_DELIVERED=$(echo $RESPONSE | jq -r '.data.last7Days.totalDelivered')
WEEK_FAILED=$(echo $RESPONSE | jq -r '.data.last7Days.totalFailed')
WEEK_RATE=$(echo $RESPONSE | jq -r '.data.last7Days.successRate')

echo "📈 LAST 7 DAYS:"
echo "   Messages Sent:      $WEEK_SENT"
echo "   Messages Delivered: $WEEK_DELIVERED"
echo "   Messages Failed:    $WEEK_FAILED"
echo "   Success Rate:       $WEEK_RATE%"
echo ""

# Extract last 30 days stats
MONTH_SENT=$(echo $RESPONSE | jq -r '.data.last30Days.totalSent')
MONTH_DELIVERED=$(echo $RESPONSE | jq -r '.data.last30Days.totalDelivered')
MONTH_FAILED=$(echo $RESPONSE | jq -r '.data.last30Days.totalFailed')
MONTH_RATE=$(echo $RESPONSE | jq -r '.data.last30Days.successRate')

echo "📅 LAST 30 DAYS:"
echo "   Messages Sent:      $MONTH_SENT"
echo "   Messages Delivered: $MONTH_DELIVERED"
echo "   Messages Failed:    $MONTH_FAILED"
echo "   Success Rate:       $MONTH_RATE%"
echo ""

# Alert if today's success rate is below 90%
if (( $(echo "$TODAY_RATE < 90" | bc -l) )); then
    echo "⚠️  ALERT: Today's success rate is below 90%!"
    echo "   Please check system status and failed messages."
fi

echo "========================================="
```

---

## Permission Matrix

| User Role | Can View All Clients | Can Filter by Client | Can View Own Client |
|-----------|---------------------|---------------------|-------------------|
| **ADMIN** | ✅ Yes (default) | ✅ Yes (via clientId param) | ✅ Yes |
| **OWNER** | ❌ No | ❌ No (ignored) | ✅ Yes (forced) |
| **STAFF** | ❌ No Access | ❌ No Access | ❌ No Access |

---

## Best Practices

1. **Regular Monitoring**
   - Check dashboard at least daily to monitor delivery health
   - Set up automated alerts for success rates below 90%

2. **Performance Tracking**
   - Compare today vs 7-day average to identify anomalies
   - Track monthly trends for capacity planning

3. **Admin Usage**
   - Use without `clientId` parameter for system-wide overview
   - Use with `clientId` parameter for client-specific troubleshooting

4. **Integration Tips**
   - Cache dashboard data for 5-10 minutes to reduce API calls
   - Use background jobs for automated monitoring
   - Create visual graphs from the statistics

5. **Interpretation**
   - Success rate > 95%: Excellent
   - Success rate 90-95%: Good
   - Success rate < 90%: Investigate immediately

6. **Troubleshooting**
   - Low success rate → Check message logs for failure reasons
   - High failed count → Check billing balance and sender configuration
   - Sudden drop in sent → Check campaign execution and scheduling

---

## Related APIs

For more detailed analytics, combine dashboard data with:

- **Message Logs API** (`/api/v1/instant-link/message-logs`) - Detailed message records
- **Billing API** (`/api/v1/instant-link/billing`) - Cost analysis
- **Campaign API** (`/api/v1/instant-link/campaigns`) - Campaign performance

---

## Data Freshness

- Dashboard queries real-time data from the database
- Statistics reflect the current state at the time of API call
- No caching is applied - every request fetches fresh data
- Time periods are calculated dynamically based on current timestamp

---

## Error Codes Summary

| Code | Message | Description |
|------|---------|-------------|
| 0 | Success | Request successful |
| 401 | Unauthorized | Missing or invalid authentication token |
| 403 | Forbidden | User doesn't have ADMIN or OWNER role |
| 500 | Internal Server Error | Database or server error occurred |

---

## Notes

1. **Message Status Definitions:**
   - **sent**: Message sent to WhatsApp API successfully
   - **delivered**: WhatsApp confirmed delivery to recipient device
   - **read**: Recipient opened/read the message
   - **failed**: Message failed to send

2. **Status Grouping:**
   - `totalDelivered` includes both "delivered" and "read" statuses
   - `totalSent` includes all outbound messages regardless of status

3. **Time Zone:**
   - All time calculations use server timezone
   - "Today" starts at 00:00:00 server time

4. **Direction Filter:**
   - Dashboard only counts `outbound` messages (sent by business)
   - `inbound` messages (received from customers) are not counted

5. **Client Filtering (ADMIN):**
   - Empty `clientId` parameter = ALL clients (aggregated)
   - Specific `clientId` parameter = That client only
   - Invalid `clientId` = Returns zero statistics (no error)

6. **Performance:**
   - Query performance depends on message volume
   - Large datasets (>1M messages) may take 1-3 seconds
   - Consider implementing frontend caching

---

**Last Updated:** December 8, 2025
**API Version:** v1
