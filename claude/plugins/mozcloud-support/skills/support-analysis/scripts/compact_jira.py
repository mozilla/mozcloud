#!/usr/bin/env python3
"""
compact_jira.py - Parse large Jira MCP responses into compact summaries.

Reads a cached Jira API response file (from an MCP tool result saved to disk)
and produces a compact summary suitable for LLM consumption. Handles both the
raw JSON response format and the text-wrapped format that MCP sometimes returns.

Usage:
    python3 compact_jira.py <response_file>
    python3 compact_jira.py <response_file> --format tickets    # One line per ticket (default)
    python3 compact_jira.py <response_file> --format stats      # Aggregate stats only
    python3 compact_jira.py <response_file> --format full       # Both tickets + stats

Input: Path to a JSON file saved by the MCP Atlassian tool when the response
       exceeded the token limit.

Output: Compact text summary to stdout.
"""

import json
import sys
import argparse
from collections import Counter
from datetime import datetime


def parse_mcp_response(filepath):
    """Parse a Jira MCP response file, handling various wrapper formats."""
    with open(filepath) as f:
        raw = json.load(f)

    # Format 1: Direct issues response
    if isinstance(raw, dict) and "issues" in raw:
        issues = raw["issues"]
        if isinstance(issues, dict):
            return issues.get("nodes", []), issues.get("totalCount", 0)
        return issues, len(issues)

    # Format 2: Text-wrapped array from MCP
    if isinstance(raw, list) and len(raw) > 0:
        if isinstance(raw[0], dict) and "text" in raw[0]:
            data = json.loads(raw[0]["text"])
            if "issues" in data:
                issues = data["issues"]
                if isinstance(issues, dict):
                    return issues.get("nodes", []), issues.get("totalCount", 0)
                return issues, len(issues)

    # Format 3: Already a list of issues
    if isinstance(raw, list) and len(raw) > 0 and "key" in raw[0]:
        return raw, len(raw)

    print(f"Warning: Could not parse response format", file=sys.stderr)
    return [], 0


def extract_ticket(node):
    """Extract key fields from a Jira issue node."""
    f = node.get("fields", {})
    key = node.get("key", "?")
    summary = (f.get("summary") or "?")[:80]
    status = (f.get("status") or {}).get("name", "?")
    priority = (f.get("priority") or {}).get("name", "None")
    project = key.split("-")[0]
    created = str(f.get("created", "?"))[:10]
    updated = str(f.get("updated", "?"))[:10]
    resdate = str(f.get("resolutiondate") or "")[:10]
    resolution = (f.get("resolution") or {}).get("name", "Open")
    assignee = (f.get("assignee") or {}).get("displayName", "Unassigned")
    reporter = (f.get("reporter") or {}).get("displayName", "?")
    issuetype = (f.get("issuetype") or {}).get("name", "?")
    comment_count = 0
    if isinstance(f.get("comment"), dict):
        comment_count = f["comment"].get("total", 0)
    links = len(f.get("issuelinks") or [])
    labels = f.get("labels") or []

    return {
        "key": key,
        "summary": summary,
        "status": status,
        "priority": priority,
        "project": project,
        "created": created,
        "updated": updated,
        "resolutiondate": resdate,
        "resolution": resolution,
        "assignee": assignee,
        "reporter": reporter,
        "issuetype": issuetype,
        "comment_count": comment_count,
        "links": links,
        "labels": labels,
    }


def print_ticket_line(t):
    """Print a single compact ticket line."""
    print(
        f"{t['key']:<16} {t['status']:<22} {t['project']:<8} "
        f"{t['created']} {t['resolution']:<12} {t['assignee']:<25} "
        f"{t['summary']}"
    )


def print_stats(tickets):
    """Print aggregate statistics."""
    statuses = Counter(t["status"] for t in tickets)
    projects = Counter(t["project"] for t in tickets)
    priorities = Counter(t["priority"] for t in tickets)
    assigned = sum(1 for t in tickets if t["assignee"] != "Unassigned")
    unassigned = len(tickets) - assigned
    reporters = Counter(t["reporter"] for t in tickets)
    types = Counter(t["issuetype"] for t in tickets)

    print()
    print("=" * 60)
    print("AGGREGATE STATISTICS")
    print("=" * 60)
    print(f"Total tickets: {len(tickets)}")
    print()
    print("PROJECTS:")
    for p, c in projects.most_common():
        print(f"  {p:<12} {c:>4}")
    print()
    print("STATUSES:")
    for s, c in statuses.most_common():
        print(f"  {s:<25} {c:>4}")
    print()
    print("PRIORITIES:")
    for p, c in priorities.most_common():
        print(f"  {p:<15} {c:>4}")
    print()
    print(f"ASSIGNED: {assigned}, UNASSIGNED: {unassigned}")
    print()
    print("TOP REPORTERS:")
    for r, c in reporters.most_common(10):
        print(f"  {r:<30} {c:>3}")
    print()
    print("ISSUE TYPES:", dict(types))

    # Needs attention: open tickets older than 7 days
    today = datetime.now().strftime("%Y-%m-%d")
    open_statuses = {"Backlog", "To Do", "Needs Clarification", "On Hold"}
    stale = [
        t for t in tickets
        if t["status"] in open_statuses and t["created"] < today
    ]
    if stale:
        print()
        print("TICKETS NEEDING ATTENTION (open in backlog/todo/NC):")
        for t in sorted(stale, key=lambda x: x["created"]):
            print(f"  {t['key']:<16} {t['status']:<22} {t['created']} {t['assignee']:<20} {t['summary']}")


def main():
    parser = argparse.ArgumentParser(description="Compact Jira MCP response")
    parser.add_argument("file", help="Path to cached MCP response JSON file")
    parser.add_argument(
        "--format", "-f",
        choices=["tickets", "stats", "full"],
        default="full",
        help="Output format (default: full)",
    )
    args = parser.parse_args()

    nodes, total = parse_mcp_response(args.file)
    tickets = [extract_ticket(n) for n in nodes]

    print(f"Parsed {len(tickets)} tickets (API reported {total} total)")

    if total > len(tickets):
        print(f"WARNING: Only {len(tickets)} of {total} tickets returned. Jira MCP has a 100-ticket limit.")
        print(f"Use nextPageToken or split the date range to get remaining tickets.")

    if args.format in ("tickets", "full"):
        print()
        print(f"{'KEY':<16} {'STATUS':<22} {'PROJ':<8} {'CREATED':<10} {'RESOLUTION':<12} {'ASSIGNEE':<25} SUMMARY")
        print("-" * 130)
        for t in tickets:
            print_ticket_line(t)

    if args.format in ("stats", "full"):
        print_stats(tickets)


if __name__ == "__main__":
    main()
