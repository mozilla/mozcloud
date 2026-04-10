# Support Analysis Scripts

Helper scripts for the `support-analysis` skill.

## Scripts

### compute_timerange.sh

Computes Unix timestamps and query filters for a given time period.

```bash
./compute_timerange.sh 30                # Last 30 days
./compute_timerange.sh 7                 # Last 7 days
./compute_timerange.sh "last 30 days"    # Natural language
./compute_timerange.sh "2 weeks"         # Weeks support
./compute_timerange.sh "March 2026"      # Full calendar month
./compute_timerange.sh "last month"      # Previous calendar month
./compute_timerange.sh "last quarter"    # Previous calendar quarter
./compute_timerange.sh "Q1 2026"         # Specific quarter
```

Outputs:
- Slack `oldest` and `latest` timestamps for channel history API
- Jira JQL filter using `labels = srein` with date range (captures tickets across all downstream projects)
- BigQuery date filter

### compact_jira.py

Parses large Jira MCP response files (saved to disk when they exceed token limits) into compact summaries.

```bash
# Full output: ticket list + aggregate stats
python3 compact_jira.py /path/to/cached-response.txt

# Just aggregate stats (projects, statuses, priorities, top reporters)
python3 compact_jira.py /path/to/cached-response.txt --format stats

# Just ticket list (one line per ticket)
python3 compact_jira.py /path/to/cached-response.txt --format tickets
```

Handles multiple MCP response formats (direct JSON, text-wrapped arrays, node arrays). Automatically detects tickets needing attention (open in Backlog/To Do/Needs Clarification).

**When to use**: When `searchJiraIssuesUsingJql` returns a response too large for the context window and saves it to a file. Pipe the file through this script instead of trying to read the raw JSON.

### extract_changelog.py

Extracts status transitions, assignee changes, and project moves from Jira issue responses that include changelog data (fetched with `expand: "changelog"`).

```bash
# Single enriched ticket response
python3 extract_changelog.py /path/to/ticket-response.txt

# Bulk response (will show which tickets have changelog data)
python3 extract_changelog.py /path/to/bulk-response.txt --bulk
```

For each ticket with changelog data, outputs:
- Status transitions with timestamps (e.g., "Backlog → In Progress on Mar 15")
- Time spent in each status (in days)
- Assignee changes (bounces)
- Project moves (SREIN → SVCSE)
- Engagement signals (3+ transitions, 2+ assignee changes, entered Needs Clarification)

**When to use**: After enriching tickets via `getJiraIssue` with `expand: "changelog"`. Parse the response through this script to get compact timelines for the engagement analysis, request quality, and SREIN ticket health sections.

## Categorization

Message categorization and cross-referencing are performed by Claude directly during the analysis, using `references/category-definitions.md` as the taxonomy. This approach is more accurate than keyword matching, especially for ambiguous messages that span multiple themes.

## Jira Pagination

The Jira MCP has a hard limit of 100 results per query. If a time period has more than 100 tickets:
1. Use `nextPageToken` from the response to fetch subsequent pages
2. Or split the date range into smaller windows (e.g., two 2-week queries for a month)
