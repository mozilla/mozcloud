# Support Analysis Scripts

Helper scripts for the `support-analysis` skill.

## Scripts

### compute_timerange.sh

Computes Unix timestamps and query filters for a given lookback period.

```bash
./compute_timerange.sh 30              # Last 30 days
./compute_timerange.sh 90              # Last 90 days (quarterly)
./compute_timerange.sh "last 7 days"   # Natural language
./compute_timerange.sh "2 weeks"       # Weeks support
```

Outputs:
- Slack `oldest` timestamp for channel history API
- Jira JQL filter using `labels = srein` (captures tickets across all downstream projects)
- BigQuery date filter

## Categorization

Message categorization and cross-referencing are performed by Claude directly during the analysis, using `references/category-definitions.md` as the taxonomy. This approach is more accurate than keyword matching, especially for ambiguous messages that span multiple themes.
