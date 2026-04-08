# Support Analysis Scripts

Helper scripts for the `support-analysis` skill. These can be used standalone or by the `customer-support-analyst` agent.

## Scripts

### compute_timerange.sh

Computes Unix timestamps and query filters for a given lookback period.

```bash
./compute_timerange.sh 30    # Last 30 days
./compute_timerange.sh 90    # Last 90 days (quarterly)
```

Outputs Slack `oldest` timestamp, Jira JQL filter, and BigQuery date filter.

### categorize_messages.py

Categorizes support messages by theme using keyword matching.

```bash
# From JSON file
python3 categorize_messages.py --file messages.json

# From stdin
cat messages.json | python3 categorize_messages.py

# Summary only (no JSON output)
python3 categorize_messages.py --file messages.json --summary-only
```

**Input**: JSON array of objects with a `text` field (Slack messages) or `summary` field (Jira tickets).

**Output**: Same JSON with `categories` and `is_automated` fields added. Summary stats printed to stderr.

### cross_reference.py

Cross-references categorized Slack and Jira data to find tracking gaps.

```bash
python3 cross_reference.py --slack slack_categorized.json --jira jira_categorized.json
```

**Output**: Table showing per-category volume in Slack vs Jira, with gap assessment and recommendations.

## Typical Workflow

```bash
# 1. Compute time range
./compute_timerange.sh 30

# 2. Export Slack messages (via MCP or API) to slack_raw.json
# 3. Export Jira tickets (via MCP or API) to jira_raw.json

# 4. Categorize both (summary goes to stderr, JSON to stdout)
python3 categorize_messages.py -f slack_raw.json > slack_categorized.json 2>/dev/null
python3 categorize_messages.py -f jira_raw.json > jira_categorized.json 2>/dev/null

# 5. Cross-reference
python3 cross_reference.py --slack slack_categorized.json --jira jira_categorized.json
```

The agent automates this entire pipeline using MCP tools for data ingestion.
