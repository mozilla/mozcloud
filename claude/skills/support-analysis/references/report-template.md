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
| Long-term engagements | |
| Needs clarification rate | |
| Top requesting service | |

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

## Engagement Analysis

### Quick Tasks vs Long-Term Engagements

| Type | Count | % of Total | Avg Duration |
|------|-------|-----------|--------------|
| Quick task (resolved ≤7 days) | | | |
| Long-term engagement (>7 days or ongoing) | | | |

### Long-Term Engagement Details

For each long-term engagement, list:

| Ticket | Summary | Status | Duration (days) | Transitions | Progressing? |
|--------|---------|--------|----------------|-------------|--------------|
| | | | | | |

### Observations
- What types of requests turn into long-term engagements?
- Are long-term engagements well-tracked or do they drift?
- Does leadership involvement (Hamid, Paul) correlate with engagement complexity?
- Recommendations for managing long-running work

---

## Request Quality

### Needs Clarification Rate

| Metric | Value |
|--------|-------|
| Total tickets | |
| Entered "Needs Clarification" | |
| Clarification rate | |
| Avg time in Needs Clarification | |

### Common Missing Information

| Missing Info Type | Occurrences | Example Ticket |
|-------------------|-------------|----------------|
| Service/environment not specified | | |
| Error details missing | | |
| Access context unclear | | |
| Vague or incomplete description | | |
| Wrong channel/form | | |

### Observations
- Patterns in which types of requests need clarification
- Whether the intake form could be improved to collect missing info upfront
- Impact of clarification delays on resolution time

---

## Requests by Service/Team

| Service/Team | Total Requests | Slack | Jira | Primary Themes | Engagement Type |
|--------------|---------------|-------|------|----------------|-----------------|
| | | | | | |

### High-Volume Services
For the top 3-5 services by request count:
- What they typically need help with
- Whether their requests are well-formed or frequently need clarification
- Quick tasks vs long-term engagements ratio
- Whether recurring issues suggest they need dedicated documentation or leveling-up support

### Observations
- Which teams are heaviest users of SRE support
- Whether certain services have recurring issues that could be addressed systemically
- Teams that may benefit from dedicated onboarding or documentation

---

## Satisfaction

### Score Distribution

| Score | Label | Count | % |
|-------|-------|-------|---|
| 5 | Delighted | | |
| 4 | Satisfied | | |
| 3 | Neutral | | |
| 2 | Frustrated | | |
| 1 | Unresolved | | |

| Metric | Value |
|--------|-------|
| Overall satisfaction rate (score 4-5) | |
| Unresolved rate (score 1) | |

### Frustrated or Unresolved Interactions
For each interaction scoring 1 or 2, list:
- Thread link or ticket key
- What the user asked for
- What went wrong (no response, bounced, escalated, etc.)

### Observations
- Are certain themes more likely to have satisfied or frustrated outcomes?
- Do certain responders consistently receive thank-you signals?
- Is there a correlation between response time and satisfaction?

---

## Recommendations

Rank by impact (high/medium/low). Each recommendation should include:
- **What**: Specific action to take
- **Why**: Evidence from the data (cite specific messages/tickets)
- **Expected impact**: Which theme it addresses and estimated reduction

Recommendations should draw from all analysis dimensions: theme breakdown, engagement patterns, request quality gaps, and per-team findings.

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

### Triage Pipeline
| Stage | Count | Avg Time in Stage |
|-------|-------|-------------------|
| SREIN (intake) → Triage | | |
| Triage → In Progress | | |
| In Progress → Done | | |
| Total created-to-resolved | | |

Note: Derived from changelog status transitions. Use `scripts/extract_changelog.py` to parse enriched ticket responses.

### Assignee Churn
| Metric | Value |
|--------|-------|
| Tickets with 0 reassignments | |
| Tickets with 1 reassignment | |
| Tickets with 2+ reassignments | |
| Avg reassignments per ticket | |

Tickets bounced between 3+ assignees are flagged as potential process issues.

### Process Observations
- Priority usage (are priorities being set?)
- Assignment coverage (% unassigned)
- Issue type differentiation
- Resolution time observations (fastest, slowest, typical)
- Time spent in Needs Clarification (from changelogs)
- Project move patterns (SREIN → SVCSE vs SREIN → MZCLD etc.)

### Tickets Needing Attention
List any tickets stuck in "Needs Clarification" or unassigned for >7 days. Link each ticket by key. Include status transition history for stalled tickets to show where they got stuck.

---

## Methodology

- **Slack source**: #mozcloud-support (C019WG3TTM2)
- **Jira source**: SREIN project
- **Time period**: [dates]
- **Bot filtering**: [describe what was filtered]
- **Limitations**: [Slack retention, heuristic categorization, DM blind spots]
