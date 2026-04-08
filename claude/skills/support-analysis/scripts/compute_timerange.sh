#!/bin/bash
#
# compute_timerange.sh - Compute Unix timestamps and JQL date filters for a given time period
#
# Usage: ./compute_timerange.sh [period]
#
# Arguments:
#   period: Flexible time expression (default: last 30 days)
#
# Output:
#   Prints Slack and Jira query parameters for the specified time range.
#
# Examples:
#   ./compute_timerange.sh                    # Last 30 days
#   ./compute_timerange.sh 7                  # Last 7 days
#   ./compute_timerange.sh "last 30 days"     # Natural language
#   ./compute_timerange.sh "2 weeks"          # Weeks support
#   ./compute_timerange.sh "March 2026"       # Full calendar month
#   ./compute_timerange.sh "last month"       # Previous calendar month
#   ./compute_timerange.sh "last quarter"     # Previous calendar quarter
#   ./compute_timerange.sh "Q1 2026"          # Specific quarter

RAW_INPUT="${*:-30}"
INPUT_LOWER=$(echo "$RAW_INPUT" | tr '[:upper:]' '[:lower:]')

# Helper: get Unix timestamp from YYYY-MM-DD (macOS + Linux)
to_epoch() {
  date -j -f "%Y-%m-%d" "$1" +%s 2>/dev/null || date -d "$1" +%s 2>/dev/null
}

# Helper: get current year/month
CURRENT_YEAR=$(date +%Y)
CURRENT_MONTH=$(date +%-m)

# Map month names to numbers
month_to_num() {
  case $(echo "$1" | tr '[:upper:]' '[:lower:]') in
    jan|january)   echo 1 ;;
    feb|february)  echo 2 ;;
    mar|march)     echo 3 ;;
    apr|april)     echo 4 ;;
    may)           echo 5 ;;
    jun|june)      echo 6 ;;
    jul|july)      echo 7 ;;
    aug|august)    echo 8 ;;
    sep|september) echo 9 ;;
    oct|october)   echo 10 ;;
    nov|november)  echo 11 ;;
    dec|december)  echo 12 ;;
    *) echo "" ;;
  esac
}

# Days in a given month/year
days_in_month() {
  local m=$1 y=$2
  case $m in
    1|3|5|7|8|10|12) echo 31 ;;
    4|6|9|11) echo 30 ;;
    2)
      if (( y % 400 == 0 )) || { (( y % 4 == 0 )) && (( y % 100 != 0 )); }; then
        echo 29
      else
        echo 28
      fi
      ;;
  esac
}

MODE=""
START_DATE=""
END_DATE=""

# --- "last month" ---
if echo "$INPUT_LOWER" | grep -qiE 'last\s+month'; then
  MODE="month"
  M=$((CURRENT_MONTH - 1))
  Y=$CURRENT_YEAR
  if [ "$M" -le 0 ]; then M=12; Y=$((Y - 1)); fi
  DIM=$(days_in_month $M $Y)
  START_DATE=$(printf "%04d-%02d-01" $Y $M)
  END_DATE=$(printf "%04d-%02d-%02d" $Y $M $DIM)

# --- "last quarter" ---
elif echo "$INPUT_LOWER" | grep -qiE 'last\s+quarter'; then
  MODE="quarter"
  CQ=$(( (CURRENT_MONTH - 1) / 3 + 1 ))
  PQ=$((CQ - 1)); PY=$CURRENT_YEAR
  if [ "$PQ" -le 0 ]; then PQ=4; PY=$((PY - 1)); fi
  SM=$(( (PQ - 1) * 3 + 1 ))
  EM=$(( PQ * 3 ))
  DIM=$(days_in_month $EM $PY)
  START_DATE=$(printf "%04d-%02d-01" $PY $SM)
  END_DATE=$(printf "%04d-%02d-%02d" $PY $EM $DIM)

# --- "Q1 2026" / "Q2 2025" etc ---
elif echo "$INPUT_LOWER" | grep -qiE '^q[1-4]\s+[0-9]{4}$'; then
  MODE="quarter"
  Q=$(echo "$INPUT_LOWER" | sed -n 's/^q\([1-4]\).*/\1/p')
  Y=$(echo "$RAW_INPUT" | grep -oE '[0-9]{4}')
  SM=$(( (Q - 1) * 3 + 1 ))
  EM=$(( Q * 3 ))
  DIM=$(days_in_month $EM $Y)
  START_DATE=$(printf "%04d-%02d-01" $Y $SM)
  END_DATE=$(printf "%04d-%02d-%02d" $Y $EM $DIM)

# --- "March 2026" / "march" / "Jan 2025" etc ---
elif MONTH_NAME=$(echo "$RAW_INPUT" | grep -oiE '(january|february|march|april|may|june|july|august|september|october|november|december|jan|feb|mar|apr|jun|jul|aug|sep|oct|nov|dec)') && [ -n "$MONTH_NAME" ]; then
  MODE="month"
  M=$(month_to_num "$MONTH_NAME")
  Y=$(echo "$RAW_INPUT" | grep -oE '[0-9]{4}')
  if [ -z "$Y" ]; then
    # No year specified — assume current year, or last year if the month is in the future
    Y=$CURRENT_YEAR
    if [ "$M" -gt "$CURRENT_MONTH" ]; then
      Y=$((Y - 1))
    fi
  fi
  DIM=$(days_in_month $M $Y)
  START_DATE=$(printf "%04d-%02d-01" $Y $M)
  END_DATE=$(printf "%04d-%02d-%02d" $Y $M $DIM)

# --- Plain integer or "N days" / "N weeks" / "last N days" ---
else
  MODE="days"
  PARSED=$(echo "$RAW_INPUT" | grep -oE '[0-9]+' | head -1)
  DAYS="${PARSED:-30}"
  if echo "$INPUT_LOWER" | grep -qiE 'weeks?'; then
    DAYS=$((DAYS * 7))
  fi
  NOW=$(date +%s)
  OLDEST=$((NOW - DAYS * 86400))
  START_DATE=$(date -r "$OLDEST" +%Y-%m-%d 2>/dev/null || date -d "@$OLDEST" +%Y-%m-%d 2>/dev/null)
  END_DATE=$(date +%Y-%m-%d)
fi

# Compute Unix timestamps
OLDEST_TS=$(to_epoch "$START_DATE")
LATEST_TS=$(to_epoch "$END_DATE")
# Add 86399 to end date to include the full last day
LATEST_TS=$((LATEST_TS + 86399))

echo "Time Range: ${START_DATE} to ${END_DATE}"
echo "============================================"
echo ""
echo "Slack parameters:"
echo "  oldest: \"${OLDEST_TS}\""
echo "  latest: \"${LATEST_TS}\""
echo "  (${START_DATE} to ${END_DATE})"
echo ""
echo "Jira JQL filter (use label, not project — tickets move to SVCSE/MZCLD after triage):"
echo "  labels = srein AND created >= \"${START_DATE}\" AND created <= \"${END_DATE}\""
echo ""
echo "BigQuery filter:"
echo "  WHERE created_at >= '${START_DATE}' AND created_at <= '${END_DATE}'"
