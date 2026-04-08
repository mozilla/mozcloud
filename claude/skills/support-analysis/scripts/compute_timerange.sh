#!/bin/bash
#
# compute_timerange.sh - Compute Unix timestamps and JQL date filters for a given lookback period
#
# Usage: ./compute_timerange.sh [period]
#
# Arguments:
#   period: Number of days or natural language (default: 30)
#
# Output:
#   Prints Slack and Jira query parameters for the specified time range.
#
# Examples:
#   ./compute_timerange.sh              # Last 30 days
#   ./compute_timerange.sh 7            # Last 7 days
#   ./compute_timerange.sh "last 30 days"  # Natural language
#   ./compute_timerange.sh "2 weeks"    # Weeks support

# Accept either a plain integer (e.g. "30") or natural language (e.g. "last 30 days", "7 days", "2 weeks")
RAW_INPUT="${*:-30}"

# Extract the first integer from the input; default to 30 if none found
PARSED=$(echo "$RAW_INPUT" | grep -oE '[0-9]+' | head -1)
DAYS="${PARSED:-30}"

# Support "week(s)" multiplier
if echo "$RAW_INPUT" | grep -qiE 'weeks?'; then
  DAYS=$((DAYS * 7))
fi

NOW=$(date +%s)
OLDEST=$((NOW - DAYS * 86400))
START_DATE=$(date -r "$OLDEST" +%Y-%m-%d 2>/dev/null || date -d "@$OLDEST" +%Y-%m-%d 2>/dev/null)
END_DATE=$(date +%Y-%m-%d)

echo "Time Range: ${DAYS} days"
echo "============================================"
echo ""
echo "Slack parameters:"
echo "  oldest: \"${OLDEST}\""
echo "  (${START_DATE} to ${END_DATE})"
echo ""
echo "Jira JQL filter:"
echo "  created >= -${DAYS}d"
echo "  (or: created >= \"${START_DATE}\")"
echo ""
echo "BigQuery filter:"
echo "  WHERE created_at >= '${START_DATE}'"
