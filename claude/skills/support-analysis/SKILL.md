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

1. **Ingest**: Run `scripts/compute_timerange.sh` to get Unix timestamps for the requested period
2. **Read Slack**: Paginate through channel history, filtering out bot/automated messages
3. **Read Jira**: Query SREIN tickets from the same period
4. **Categorize**: Assign messages and tickets to theme buckets using `references/category-definitions.md`
5. **Cross-reference**: Compare Slack volume vs Jira volume per theme
6. **Report**: Output using `references/report-template.md` structure

## Output

The report includes:
- Executive summary with key metrics
- Theme breakdown with Slack vs Jira counts and tracking gaps
- Cross-reference analysis (well-tracked vs invisible themes)
- Top requesters
- Actionable recommendations ranked by impact
- SREIN ticket status and process observations

## Related Agent

The `customer-support-analyst` agent at `claude/agents/customer-support-analyst.md` is the autonomous counterpart to this skill. It contains the full analysis framework and can be launched for end-to-end report generation.
