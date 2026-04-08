---
name: support-analysis
description: Analyze MozCloud customer support patterns across Slack (#mozcloud-support) and Jira (SREIN), categorize themes, cross-reference sources, and produce actionable reports
user-invocable: true
disable-model-invocation: false
allowed-tools: Read, Grep, Glob, Bash, mcp__plugin_slack_slack__slack_read_channel, mcp__plugin_slack_slack__slack_read_thread, mcp__plugin_slack_slack__slack_read_user_profile, mcp__plugin_slack_slack__slack_search_channels, mcp__plugin_slack_slack__slack_search_public, mcp__plugin_slack_slack__slack_search_public_and_private, mcp__plugin_slack_slack__slack_search_users, mcp__atlassian__searchJiraIssuesUsingJql, mcp__atlassian__getJiraIssue, mcp__atlassian__createConfluencePage, mcp__atlassian__updateConfluencePage, mcp__atlassian__getConfluenceSpaces, mcp__atlassian__getConfluencePage, mcp__atlassian__atlassianUserInfo
---

## Requirements

This skill requires:
- **Slack MCP server** — for reading #mozcloud-support channel history
- **Atlassian MCP server** — for querying SREIN Jira tickets

If either is unavailable, run `/mcp` to reconnect.

## General

Analyze MozCloud customer support interactions to identify patterns, pain points, and improvement opportunities. The skill reads from two sources:

1. **Slack #mozcloud-support** (channel ID: `C019WG3TTM2`) — real-time support conversations
2. **Jira tickets with `srein` label** — formal support tickets across multiple projects

SREIN tickets are triaged and moved to downstream projects (SVCSE, MZCLD, INFRASEC, etc.) while retaining the `srein` label. Always query by label, not project, to get the full picture:
- `labels = srein AND created >= "YYYY-MM-DD"` — captures all tickets regardless of current project
- `project = SREIN AND created >= "YYYY-MM-DD"` — only returns untriaged tickets still in the intake queue

The analysis cross-references both sources to find tracking gaps where support work happens in Slack but is never captured in Jira.

## Workflow

1. **Compute time range**: Run `scripts/compute_timerange.sh` to get Unix timestamps and JQL date filters
2. **Read Slack**: Paginate through #mozcloud-support channel history using `oldest` timestamp. Filter out automated messages (Jira bot user `U8TT4054G`, SRE BOT triage announcements, empty Grafana alerts, channel management messages)
3. **Read Jira**: Query `labels = srein AND created >= "YYYY-MM-DD"` with maxResults=100. Parse the response to extract key, summary, status, priority, assignee, project, and resolution fields
4. **Categorize**: Classify each message and ticket into themes per the instructions below
5. **Cross-reference**: Compare Slack volume vs Jira volume per theme to identify tracking gaps
6. **Report**: Output using `references/report-template.md` structure

## Classification

Read `references/category-definitions.md` for the full taxonomy. Classify each Slack message and Jira ticket into one or more of the 12 defined themes.

### How to classify

- **Read the full message**, not just keywords. A message mentioning "helm" in the context of a deploy failure is `deployment-argocd`, not `helm-k8s`. A message asking "how do I configure HPA" is `helm-k8s`, possibly also `documentation`.
- **Assign multiple categories** when a message genuinely spans themes. A user asking for Grafana access is both `access-permissions` and `monitoring-grafana`. A deploy failure that becomes an outage is both `deployment-argocd` and `incident`.
- **Use `uncategorized` sparingly**. Most messages fit at least one theme. FYI announcements, social messages, and off-topic conversations are uncategorized. If you're unsure, pick the closest theme rather than defaulting to uncategorized.
- **Distinguish requests from discussions**. A PR link with `:review:` is `pr-review`. A message discussing the same PR's terraform plan failure is `terraform-atlantis`. Classify by what the person is asking for, not what the PR contains.

### Filtering automated messages

Before classifying, separate automated messages from human ones. Count both but only classify human messages into themes. Automated messages include:
- **Jira bot** (user ID `U8TT4054G`): Ticket creation notifications posted to channel. Identified by user ID or by content matching "*[Name] created a Task*" with an SREIN link.
- **SRE BOT**: Daily triage rotation announcements starting with "CloudEng Triage Response Update"
- **Grafana alerts**: Empty or near-empty messages from Grafana
- **Channel management**: "has joined the channel", "set the channel topic", "pinned a message"

### Counting rules

- Count each **top-level Slack message** once. Do not count thread replies (they aren't returned by the channel history API).
- If a single message maps to multiple categories, count it once per category for the theme breakdown, but count it as one message for the total.
- For Jira tickets, classify by the **ticket summary**. If the summary is ambiguous, use the downstream project as a signal (e.g., INFRASEC tickets are often `access-permissions` or `dns-networking`).
- When a user posts multiple messages as part of the same conversation (e.g., follow-up in the same minute), count them as one interaction for the top requesters table.

### Edge cases

- **Austin Mitchell's OpenTofu FYI announcements**: These are human messages but informational, not support requests. Classify as `terraform-atlantis` and note them as FYI/coordination rather than support questions.
- **Incident escalations**: A message like "Do we have SREs that can help?" is `incident` even if it doesn't use the word "incident". Look for urgency, production impact, and cross-team escalation signals.
- **"Can I..." questions about GCP/billing**: These are `gcp-config` even when phrased as access questions, unless the user is specifically asking to be added to a role or group.
- **PR reviews that mention urgency** (`:priority-highest:`, "revenue impacting", "time-sensitive"): Still `pr-review`, but flag these in the theme details as evidence that the review process lacks prioritization.

## Output

The report includes:
- Executive summary with key metrics
- Theme breakdown with Slack vs Jira counts and tracking gaps
- Cross-reference analysis (well-tracked vs invisible themes)
- Top requesters
- Actionable recommendations ranked by impact
- SREIN ticket status and process observations

## Linking Jira Issues

When producing reports (especially for Confluence), always link Jira ticket keys to their browse URL: `[SREIN-123](https://mozilla-hub.atlassian.net/browse/SREIN-123)`. This applies to theme details, tickets needing attention, and any ticket referenced in the report. In Confluence's Atlassian storage format, use the Jira issue macro or a plain `<a>` link — Confluence auto-renders linked ticket keys.

## Publishing to Confluence

After creating or updating a Confluence page, set it to **full-width layout** by setting two content properties via the REST API:

1. `content-appearance-published` → `"full-width"`
2. `content-appearance-draft` → `"full-width"`

For each property, GET `/wiki/rest/api/content/{pageId}/property/{key}` first:
- If it exists (200), PUT with the value and an incremented `version.number`
- If it doesn't exist (404), POST to `/wiki/rest/api/content/{pageId}/property` with the key and value

**Limitation**: The Atlassian MCP server does not currently expose content property APIs or raw REST access. Until it does, prompt the user to manually set the page to full-width via the Confluence UI (page menu → "Full width").

