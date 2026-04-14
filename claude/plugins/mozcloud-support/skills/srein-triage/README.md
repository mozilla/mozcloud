# SREIN Triage Skill

A Claude Code skill that triages JIRA customer support requests in the MozCloud Intake (SREIN) project. It fetches backlog issues, reads the current triage process from Confluence, and produces an HTML report with structured triage suggestions.

## Prerequisites

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed
- Access to `mozilla-hub.atlassian.net` (JIRA and Confluence)

## Setup

### 1. Install the Atlassian MCP plugin

The skill uses the [Atlassian MCP server](https://github.com/atlassian/atlassian-mcp-server) to access JIRA and Confluence. Install it from the Claude Code plugin marketplace:

1. Open Claude Code
2. Run `/plugins`
3. Search for **atlassian** and install it
4. Follow the OAuth prompt to authenticate with your `mozilla-hub.atlassian.net` account in your browser

### 2. Install the mozcloud-support plugin

```bash
claude plugin add-marketplace https://github.com/mozilla/mozcloud
claude plugin install mozcloud-support
```

### 3. Add recommended permissions

To avoid being prompted for every JIRA/Confluence API call, add these permissions to your project's `.claude/settings.local.json` (in the repo where you'll run triage):

```json
{
  "permissions": {
    "allow": [
      "mcp__atlassian__getAccessibleAtlassianResources",
      "mcp__atlassian__searchJiraIssuesUsingJql",
      "mcp__atlassian__getJiraIssue",
      "mcp__atlassian__getConfluencePage",
      "mcp__atlassian__searchConfluenceUsingCql"
    ]
  }
}
```

These are read-only permissions. The skill never modifies tickets or Confluence pages.

## Usage

Run the skill in Claude Code:

```
/srein-triage
```

This triages all issues in Backlog and Needs Clarification status.

To triage a single issue:

```
/srein-triage SREIN-1127
```

## Output

The skill writes an HTML report to `srein_triage/srein_YYYYMMDD_triage_report.html` in the current working directory. Open it in a browser to review the triage suggestions.

The report includes:
- Issues grouped by status (Backlog first, then Needs Clarification)
- Color-coded cards (blue for Backlog, orange for Needs Clarification, red for stale NC issues)
- Structured meta table per issue (reporter with intake history and team affiliation, dates, priority)
- Links extracted from issue descriptions, comments, and JIRA linked issues
- 7-step triage assessment for each issue with pass/fail indicators
- Priority recommendation (P0/P1/P2) based on urgency and deadlines
- Relevant CloudEng documentation from the SRE Confluence space (runbooks, how-to guides)
- Actionable summary recommendation for each issue

## Directory structure

```
srein-triage/
  SKILL.md                              # Skill definition and triage process
  README.md                             # This file
  templates/
    report_template.html                # HTML/CSS template for the triage report
  scripts/
    extract_issues.py                   # Parse saved JQL JSON into readable issue summaries
    extract_project_breakdown.py        # Extract project-key frequency for reporter affiliation
  evals/
    evals.json                          # Skill evaluation definitions
```

## How it works

1. Fetches the current MozCloud triage process and FAQ from Confluence
2. Queries JIRA for SREIN issues in Backlog and Needs Clarification status
3. For each issue, gathers context: linked issues, reporter intake history, team affiliation, and relevant CloudEng documentation from the SRE Confluence space
4. Applies the 7-step triage process:
   1. Request clarity
   2. Self-serve check
   3. Cost-savings labels
   4. Priority recommendation (P0/P1/P2)
   5. Platform improvement opportunity
   6. Can triage do it directly
   7. Routing to appropriate backlog
5. Writes the HTML report

The skill is suggestion-only -- it never modifies tickets, adds comments, or changes status.

## Known limitations

### Context window pressure with large backlogs

The skill fetches two large Confluence pages plus all issue data (descriptions, comments, linked issues, reporter history) into the LLM's context window. With 6-10 issues this works well. With 20+ issues, later batches may receive shallower analysis as the context fills up. If you notice quality degradation on later issues, consider running the skill on specific issues instead of the full backlog.

### API call volume from reporter context

The skill runs 2 extra JQL queries per unique reporter (intake history + team affiliation) plus 1 CQL query per issue (CloudEng documentation search). For a batch of 10 issues with 10 unique reporters, that's 30 additional API calls on top of the issue and linked-issue fetches. This hasn't been a problem so far, but if it becomes slow, possible mitigations include:
- Caching reporter context in a local file between runs (reporters don't change teams often)
- Reducing `maxResults` on the team affiliation query
- Dropping the team affiliation query and inferring team from ticket content and email domain

### Hardcoded Confluence page IDs

The skill references two Confluence pages by ID:
- `1560838234` -- "MozCloud Intake and Triage"
- `2439807017` -- "MozCloud Triage FAQ"

If these pages are moved, renamed, or replaced, the skill will warn that the fetched titles don't match. Update the page IDs in `SKILL.md` if this happens.
