# mozcloud-support

Claude Code plugin for MozCloud customer support analysis.

## What's included

| Type | Name | Description |
|------|------|-------------|
| Skill | `support-analysis` | Analyze support patterns across Slack (#mozcloud-support) and Jira (SREIN), categorize themes, and produce actionable reports |

## Requirements

None — this is a pure skills plugin with no binary dependencies.

## Install

```bash
# One-time: add the mozcloud marketplace
claude plugin add-marketplace mozilla/mozcloud

# Install this plugin
claude plugin install mozcloud-support
```

## Usage

Once installed, use the `/support-analysis` skill or ask Claude to analyze recent support activity.
