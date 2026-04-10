# Report Template

Use this structure for all support analysis reports.

---

# MozCloud Support Analysis — [Time Period]

## Executive Summary

| Metric | Value |
|--------|-------|
| Total Slack messages | |
| Human messages | |
| Automated/bot messages | |
| SREIN tickets | |
| Ticket-to-Slack ratio | |
| SREIN resolution rate | |

**Key findings:** (3-5 bullet points summarizing the most important insights)

---

## Theme Breakdown

| Theme | Slack | SREIN | Tracking Gap |
|-------|-------|-------|--------------|
| access-permissions | | | |
| deployment-argocd | | | |
| terraform-atlantis | | | |
| dns-networking | | | |
| monitoring-grafana | | | |
| database | | | |
| gcp-config | | | |
| helm-k8s | | | |
| incident | | | |
| onboarding | | | |
| pr-review | | | |
| documentation | | | |

### Theme Details

For each theme with significant volume, include:
- Count and trend (increasing/decreasing/stable)
- 2-3 representative message excerpts with dates and users
- Whether resolution typically happens in Slack or Jira
- Any systemic patterns (same question from multiple users = doc gap)
- Related Jira tickets linked by key (e.g., [SREIN-123](https://mozilla-hub.atlassian.net/browse/SREIN-123))

---

## Cross-Reference Analysis

### Well-Tracked (Slack → Jira pipeline works)
List themes where SREIN tickets are proportional to Slack volume.

### Under-Tracked (Slack-heavy, Jira-light)
List themes with many Slack messages but few/no tickets. Explain impact.

### Invisible Work (No record)
Themes resolved entirely in Slack threads or DMs. Estimate SRE time consumed.

---

## Top Requesters

| User | Interactions | Primary Themes |
|------|-------------|----------------|
| | | |

Note any users who appear to be struggling with onboarding or repeated issues.

---

## Recommendations

Rank by impact (high/medium/low). Each recommendation should include:
- **What**: Specific action to take
- **Why**: Evidence from the data (cite specific messages/tickets)
- **Expected impact**: Which theme it addresses and estimated reduction

### High Impact
1.
2.

### Medium Impact
1.
2.

### Low Impact
1.

---

## SREIN Ticket Health

### Status Distribution
| Status | Count | % |
|--------|-------|---|
| Done | | |
| In Progress | | |
| Needs Clarification | | |
| Backlog | | |
| Cancelled | | |

### Process Observations
- Priority usage (are priorities being set?)
- Assignment coverage (% unassigned)
- Issue type differentiation
- Resolution time observations (fastest, slowest, typical)

### Tickets Needing Attention
List any tickets stuck in "Needs Clarification" or unassigned for >7 days. Link each ticket by key.

---

## Methodology

- **Slack source**: #mozcloud-support (C019WG3TTM2)
- **Jira source**: SREIN project
- **Time period**: [dates]
- **Bot filtering**: [describe what was filtered]
- **Limitations**: [Slack retention, heuristic categorization, DM blind spots]
