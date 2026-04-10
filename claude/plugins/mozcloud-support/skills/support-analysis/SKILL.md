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

### Execution strategy

Use **parallel background agents** for data gathering to minimize latency. After computing the time range (step 1), launch steps 2 and 3 as simultaneous background agents. Once both complete, run the analysis steps (4-5) either inline or via a synthesis agent, then publish.

Recommended agent split:
- **Agent 1 (Slack)**: Steps 2, 5a — paginate all channel messages, then read all threads for satisfaction signals
- **Agent 2 (Jira)**: Steps 3, 3a, 5c — bulk query, run `compact_jira.py`, query NC history (`status WAS`), enrich key tickets with changelogs via `extract_changelog.py`
- **Agent 3 (Synthesis)**: Steps 4, 5, 5b, 5d, 5e, 6 — receives data from agents 1+2, classifies, cross-references, scores, produces report

For smaller time windows (≤7 days), agents 1 and 2 are sufficient — synthesis can be done inline. For larger windows (month, quarter), a dedicated synthesis agent keeps context cleaner.

### Steps

1. **Compute time range**: Run `scripts/compute_timerange.sh` to get Unix timestamps and JQL date filters
2. **Read Slack**: Paginate through #mozcloud-support channel history using `oldest` and `latest` timestamps. Filter out automated messages (Jira bot user `U8TT4054G`, SRE BOT triage announcements, empty Grafana alerts, channel management messages)
3. **Read Jira**: Query `labels = srein AND created >= "YYYY-MM-DD" AND created <= "YYYY-MM-DD"` with maxResults=100. Request fields: `summary, status, priority, assignee, project, resolution, created, updated, resolutiondate, description, comment, reporter, issuelinks, issuetype, labels`. The Jira MCP has a hard limit of 100 results per query. If the response indicates more tickets exist (totalCount > returned count), **paginate** using `nextPageToken` or split the date range into smaller windows (e.g., two 2-week queries instead of one month). When the MCP response is too large and gets saved to a file, use `scripts/compact_jira.py <filepath>` to parse it into a compact summary instead of trying to read the raw JSON.
3a. **Enrich tickets with changelogs**: The bulk search does not return status transition history. For each ticket, fetch full details via `getJiraIssue` with `expand: "changelog"` and `responseContentFormat: "markdown"` to capture the complete state change timeline. To manage volume, batch this efficiently:
   - Use `scripts/compact_jira.py` to parse the bulk response and identify which tickets need enrichment
   - At minimum, enrich tickets that are: (a) still open, (b) in or passed through "Needs Clarification", (c) resolved with duration >7 days, or (d) have 3+ comments suggesting back-and-forth
   - For remaining tickets (quick Done tasks), the bulk response fields are sufficient
   - From the changelog, extract: status transitions with timestamps (e.g., "Backlog → In Progress on Mar 15, In Progress → Done on Mar 17"), assignee changes, and time spent in each status. This data feeds the engagement analysis, request quality assessment, and satisfaction scoring.
   - When enriched ticket responses are saved to files, use `scripts/extract_changelog.py <filepath>` to produce compact timelines.
4. **Categorize**: Classify each message and ticket into themes per the instructions below
5. **Cross-reference**: Compare Slack volume vs Jira volume per theme to identify tracking gaps
5a. **Read threads**: For each human Slack message that has replies (indicated by "Thread: N replies" in channel history), use `slack_read_thread` to read the full thread. Note resolution signals (thanks, positive reactions, silence, escalation) for satisfaction scoring.
5b. **Classify engagement type**: For each Jira ticket, classify as "quick task" or "long-term engagement" per the engagement analysis instructions below
5c. **Assess request quality**: Query `labels = srein AND created >= "YYYY-MM-DD" AND created <= "YYYY-MM-DD" AND status WAS "Needs Clarification"` to find ALL tickets that passed through NC — not just those currently in NC. The `status WAS` JQL operator captures tickets that transitioned through NC and moved on (to Done, Cancelled, etc.). Analyze per the request quality instructions below.
5d. **Group by team/service**: Extract the service or team from each ticket and Slack message per the team grouping instructions below
5e. **Score satisfaction**: Assign a 1-5 satisfaction score to each interaction (Slack thread + Jira ticket) per the satisfaction scoring instructions below
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

## Engagement Analysis

Classify each Jira ticket as **quick task** or **long-term engagement**. Use the following signals (any one is sufficient, but multiple reinforce the classification):

- **Duration**: Created-to-resolved >7 days, or still open >14 days
- **Status transitions**: 3+ transitions in the changelog suggest iterative work, not a simple request
- **Summary patterns**: Keywords like "migration", "onboarding", "decommission", "integration", "new tenant", "new service", "setup" suggest long-term engagements
- **Issue links**: 2+ linked issues suggest the ticket is part of a larger effort
- **Leadership involvement**: If Hamid Tahsildoost (htahsildoost@mozilla.com, Director Cloud Eng) or Paul Hammer (phammer@mozilla.com, Cloud Eng Manager) appear in ticket comments, flag as likely long-term — their involvement signals escalation and coordination beyond a routine task
- **Issue type**: Epics with the `srein` label are definitionally long-term, but this pattern is rare

Note: The SREIN→SVCSE→Epic-with-subtasks pattern is not in active use. Long-term detection should rely primarily on duration, summary patterns, and leadership comment involvement rather than issue hierarchy.

For the report, summarize: total quick vs long-term count, what happened to the long-term ones (resolved, still open, stalled), and average duration for each type. List each long-term engagement with its ticket key, summary, status, duration, and whether it's progressing or stalled.

## Request Quality

For tickets that entered "Needs Clarification" status at any point — use `status WAS "Needs Clarification"` in the JQL, not just current status. Many tickets pass through NC and move on to Done or Cancelled, so checking only current status dramatically undercounts.

1. **Count**: How many tickets passed through "Needs Clarification" out of the total? Compute the clarification rate. Break down by outcome: NC→Done (clarified and resolved), NC→Cancelled (misdirected or abandoned), NC→still open.
2. **Read descriptions**: For each Needs Clarification ticket, read the description and identify what information was missing. Common categories:
   - Service or environment not specified
   - Error details or reproduction steps missing
   - Access context unclear (who needs access to what, and why)
   - Vague or incomplete request description
   - Wrong channel or form used
3. **Read comments**: Read the first 1-2 comments on each Needs Clarification ticket to see what the assignee asked for. This reveals the gap between what the form collects and what SREs need.
4. **Time in limbo**: From the changelog, note how long tickets sat in Needs Clarification before moving forward.
5. **Summarize**: Group the missing-info patterns, note which patterns are most common, and suggest intake form improvements.

## Team/Service Grouping

For each Jira ticket and Slack message, extract the **service or team** the request relates to:

1. **Extract from text**: Look for known service names in the summary or message:
   Autopush, Merino, AMO (addons-server), Lando, Guardian, Pocket, MDN, Firefox Accounts (FxA), Firefox Sync, Bedrock (mozilla.org), Taskcluster, Treeherder, Perfherder, Phabricator, Bugzilla, Glean, Crash Stats (Socorro), Relay, Monitor, VPN, Rally, Contile, SUMO, Pontoon, Springfield, Solo/Soloist
2. **Fall back to reporter**: If no service name is identifiable, use the reporter's displayName. For obvious email domain mappings (e.g., @getpocket.com = Pocket team), note the team.
3. **Fall back to "unspecified"**: If neither text nor reporter reveals the team.
4. **Produce a table**: Service/team → total request count, Slack vs Jira split, primary themes, and whether requests tend to be quick tasks or long-term engagements.
5. **Narrative for top services**: For the top 3-5 services by request count, describe what they typically need help with, whether their requests are well-formed, and whether they have recurring issues that suggest they need dedicated documentation or leveling-up support.

## Satisfaction Scoring

For each human Slack thread and each Jira ticket, assign a **satisfaction score (1-5)** based on resolution signals. This requires reading thread replies via `slack_read_thread` for every human message that has replies (check the "Thread: N replies" indicator in channel history).

### Scoring rubric

| Score | Label | Signals |
|-------|-------|---------|
| 5 | Delighted | Explicit thanks + positive language ("perfect", "awesome", "lifesaver", "exactly what I needed"), or thank-you/prayer-hands emoji reactions on the resolution message |
| 4 | Satisfied | "thanks", "thank you", "that worked", checkmark/thumbsup reaction on resolution, or Jira resolved as Done within 3 days |
| 3 | Neutral | Thread appears resolved (topic moved on, or solution was provided) but no explicit acknowledgment from the requester. Jira resolved as Done but took >7 days. |
| 2 | Frustrated | User had to re-ask the same question, thread went silent after an SRE response with no confirmation it helped, ticket bounced between 3+ assignees, or user escalated to a different channel |
| 1 | Unresolved | Thread got zero responses, ticket stuck in Needs Clarification with no requester follow-up, or ticket Cancelled after prolonged inactivity |

### How to score

- **Slack threads**: Use `slack_read_thread` for each human message that has replies. Read the full thread and look for resolution signals from the original requester (not the responder). Score based on the strongest signal present.
- **Jira tickets**: Score based on resolution status, time to resolve, number of assignee changes (from changelog), and whether the ticket passed through Needs Clarification.
- **Slack messages with no thread**: If a human message got no replies at all, score as 1 (Unresolved). If it was a `:review:` PR request that doesn't need a thread reply (resolution happens in GitHub), score as 3 (Neutral) by default.
- **Cross-reference**: If the same interaction appears in both Slack and Jira, use the higher of the two scores.

### Reactions as signal

Slack reactions on messages are visible in the channel history. Key reaction signals:
- Positive: `thumbsup`, `white_check_mark`, `heavy_check_mark`, `pray`, `raised_hands`, `tada`, `heart`
- Acknowledgment: `eyes` (someone saw it, but not a satisfaction signal)
- Negative: None reliably — absence of positive reactions on a resolved thread is a neutral signal, not negative

### Report output

Produce a summary table:

| Score | Label | Count | % |
|-------|-------|-------|---|
| 5 | Delighted | | |
| 4 | Satisfied | | |
| 3 | Neutral | | |
| 2 | Frustrated | | |
| 1 | Unresolved | | |

Also compute:
- **Overall satisfaction rate**: % of interactions scoring 4 or 5
- **Unresolved rate**: % of interactions scoring 1
- Flag any interactions scoring 1 or 2 with the ticket/thread link and a brief note on what went wrong

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

