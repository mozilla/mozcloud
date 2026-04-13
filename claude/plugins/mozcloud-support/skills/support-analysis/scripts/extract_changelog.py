#!/usr/bin/env python3
"""
extract_changelog.py - Extract status transitions and assignee changes from Jira issue responses.

Parses a single Jira issue response (from getJiraIssue with expand=changelog) and produces
a compact timeline of state changes. Can also process the bulk search response and extract
whatever changelog data is available.

Usage:
    # Single enriched ticket (from getJiraIssue with expand=changelog)
    python3 extract_changelog.py <response_file>

    # Bulk response - extract available history from all tickets
    python3 extract_changelog.py <response_file> --bulk

Output: Compact timeline per ticket showing status transitions, assignee changes, and
        time spent in each status.
"""

import json
import sys
import argparse
from datetime import datetime, timedelta
from collections import defaultdict


def parse_response(filepath):
    """Parse a Jira MCP response file."""
    with open(filepath) as f:
        raw = json.load(f)

    # Single issue response
    if isinstance(raw, dict) and "key" in raw:
        return [raw]

    # Text-wrapped from MCP
    if isinstance(raw, list) and len(raw) > 0 and isinstance(raw[0], dict) and "text" in raw[0]:
        data = json.loads(raw[0]["text"])
        if "key" in data:
            return [data]
        if "issues" in data:
            issues = data["issues"]
            return issues.get("nodes", issues) if isinstance(issues, dict) else issues

    # Bulk search response
    if isinstance(raw, dict) and "issues" in raw:
        issues = raw["issues"]
        return issues.get("nodes", issues) if isinstance(issues, dict) else issues

    # Already a list of issues
    if isinstance(raw, list) and len(raw) > 0 and "key" in raw[0]:
        return raw

    return []


def extract_status_transitions(changelog):
    """Extract status transitions from a changelog."""
    transitions = []
    if not changelog:
        return transitions

    histories = changelog.get("histories", [])
    for history in histories:
        created = history.get("created", "")[:19]
        author = (history.get("author") or {}).get("displayName", "?")
        for item in history.get("items", []):
            if item.get("field") == "status":
                transitions.append({
                    "timestamp": created,
                    "from": item.get("fromString", "?"),
                    "to": item.get("toString", "?"),
                    "author": author,
                })
    return sorted(transitions, key=lambda x: x["timestamp"])


def extract_assignee_changes(changelog):
    """Extract assignee changes from a changelog."""
    changes = []
    if not changelog:
        return changes

    histories = changelog.get("histories", [])
    for history in histories:
        created = history.get("created", "")[:19]
        for item in history.get("items", []):
            if item.get("field") == "assignee":
                changes.append({
                    "timestamp": created,
                    "from": item.get("fromString") or "Unassigned",
                    "to": item.get("toString") or "Unassigned",
                })
    return sorted(changes, key=lambda x: x["timestamp"])


def extract_project_changes(changelog):
    """Extract project moves (SREIN -> SVCSE etc) from a changelog."""
    moves = []
    if not changelog:
        return moves

    histories = changelog.get("histories", [])
    for history in histories:
        created = history.get("created", "")[:19]
        for item in history.get("items", []):
            if item.get("field") == "project":
                moves.append({
                    "timestamp": created,
                    "from": item.get("fromString", "?"),
                    "to": item.get("toString", "?"),
                })
    return sorted(moves, key=lambda x: x["timestamp"])


def compute_time_in_status(transitions, created_date, current_status):
    """Compute time spent in each status."""
    durations = defaultdict(float)
    if not transitions:
        return durations

    # Start from creation in first status
    prev_status = transitions[0]["from"] if transitions else current_status
    prev_time = created_date

    for t in transitions:
        try:
            t_time = datetime.fromisoformat(t["timestamp"].replace("Z", "+00:00")).replace(tzinfo=None)
        except (ValueError, TypeError):
            continue
        delta = (t_time - prev_time).total_seconds() / 86400  # days
        if delta > 0:
            durations[prev_status] += delta
        prev_status = t["to"]
        prev_time = t_time

    # Time in current status
    now = datetime.now()
    delta = (now - prev_time).total_seconds() / 86400
    if delta > 0:
        durations[prev_status] += delta

    return durations


def print_ticket_timeline(issue):
    """Print compact timeline for a single issue."""
    key = issue.get("key", "?")
    fields = issue.get("fields", {})
    summary = (fields.get("summary") or "?")[:70]
    status = (fields.get("status") or {}).get("name", "?")
    created_str = str(fields.get("created", "?"))[:10]
    changelog = issue.get("changelog")

    print(f"\n{'=' * 80}")
    print(f"{key}: {summary}")
    print(f"Current status: {status} | Created: {created_str}")
    print(f"{'=' * 80}")

    if not changelog:
        print("  (no changelog data — ticket was not fetched with expand=changelog)")
        return

    # Status transitions
    transitions = extract_status_transitions(changelog)
    if transitions:
        print(f"\n  Status transitions ({len(transitions)}):")
        for t in transitions:
            ts = t["timestamp"][:10]
            print(f"    {ts}  {t['from']} → {t['to']}")

        # Time in each status
        try:
            created_dt = datetime.fromisoformat(str(fields.get("created", ""))[:19])
        except (ValueError, TypeError):
            created_dt = datetime.now()
        durations = compute_time_in_status(transitions, created_dt, status)
        if durations:
            print(f"\n  Time in each status:")
            for s, days in sorted(durations.items(), key=lambda x: -x[1]):
                print(f"    {s:<25} {days:.1f} days")
    else:
        print("\n  No status transitions (ticket may have been resolved without status changes)")

    # Assignee changes
    assignee_changes = extract_assignee_changes(changelog)
    if assignee_changes:
        print(f"\n  Assignee changes ({len(assignee_changes)}):")
        for c in assignee_changes:
            ts = c["timestamp"][:10]
            print(f"    {ts}  {c['from']} → {c['to']}")

    # Project moves
    project_moves = extract_project_changes(changelog)
    if project_moves:
        print(f"\n  Project moves ({len(project_moves)}):")
        for m in project_moves:
            ts = m["timestamp"][:10]
            print(f"    {ts}  {m['from']} → {m['to']}")

    # Summary signals for engagement detection
    n_transitions = len(transitions)
    n_assignees = len(assignee_changes)
    needs_clar = any(t["to"] == "Needs Clarification" for t in transitions)
    signals = []
    if n_transitions >= 3:
        signals.append(f"{n_transitions} status transitions")
    if n_assignees >= 2:
        signals.append(f"{n_assignees} assignee changes (bounced)")
    if needs_clar:
        signals.append("entered Needs Clarification")
    if signals:
        print(f"\n  Engagement signals: {', '.join(signals)}")


def main():
    parser = argparse.ArgumentParser(description="Extract Jira changelog timelines")
    parser.add_argument("file", help="Path to Jira response JSON file")
    parser.add_argument("--bulk", action="store_true",
                        help="Process as bulk search response (multiple tickets)")
    args = parser.parse_args()

    issues = parse_response(args.file)

    if not issues:
        print("No issues found in response", file=sys.stderr)
        sys.exit(1)

    has_changelog = sum(1 for i in issues if i.get("changelog"))
    print(f"Parsed {len(issues)} issue(s), {has_changelog} with changelog data")

    for issue in issues:
        print_ticket_timeline(issue)


if __name__ == "__main__":
    main()
