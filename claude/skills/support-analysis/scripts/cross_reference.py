#!/usr/bin/env python3
"""
cross_reference.py - Cross-reference Slack messages with Jira tickets by category.

Usage:
    python3 cross_reference.py --slack slack_categorized.json --jira jira_categorized.json

Input: Two JSON files from categorize_messages.py output (with "categories" field).

Output: Cross-reference report showing Slack vs Jira volume per category,
        tracking gaps, and recommendations.
"""

import json
import argparse
from collections import Counter


def load_human_messages(filepath):
    """Load and filter to human-only messages."""
    with open(filepath) as f:
        data = json.load(f)
    return [m for m in data if not m.get("is_automated", False)]


def count_categories(messages):
    """Count category occurrences across messages."""
    counts = Counter()
    for msg in messages:
        for cat in msg.get("categories", ["uncategorized"]):
            counts[cat] += 1
    return counts


def classify_gap(slack_count, jira_count):
    """Classify the tracking gap between Slack and Jira."""
    if slack_count == 0 and jira_count == 0:
        return "No activity"
    if jira_count == 0 and slack_count > 0:
        return "CRITICAL — Zero tickets"
    ratio = jira_count / slack_count if slack_count > 0 else float("inf")
    if ratio >= 0.3:
        return "Well-tracked"
    elif ratio >= 0.1:
        return "Under-tracked"
    else:
        return "HIGH — Mostly invisible"


ALL_CATEGORIES = [
    "access-permissions",
    "deployment-argocd",
    "terraform-atlantis",
    "dns-networking",
    "monitoring-grafana",
    "database",
    "gcp-config",
    "helm-k8s",
    "incident",
    "onboarding",
    "pr-review",
    "documentation",
]


def main():
    parser = argparse.ArgumentParser(description="Cross-reference Slack and Jira support data")
    parser.add_argument("--slack", required=True, help="Categorized Slack messages JSON")
    parser.add_argument("--jira", required=True, help="Categorized Jira tickets JSON")
    args = parser.parse_args()

    slack_msgs = load_human_messages(args.slack)
    jira_msgs = load_human_messages(args.jira)

    slack_counts = count_categories(slack_msgs)
    jira_counts = count_categories(jira_msgs)

    print("CROSS-REFERENCE: Slack vs SREIN Jira")
    print("=" * 75)
    print(f"{'Category':<25} {'Slack':>6} {'SREIN':>6} {'Ratio':>8}  Gap Assessment")
    print("-" * 75)

    well_tracked = []
    under_tracked = []
    invisible = []

    for cat in ALL_CATEGORIES:
        s = slack_counts.get(cat, 0)
        j = jira_counts.get(cat, 0)
        ratio = f"{j/s:.0%}" if s > 0 else "N/A"
        gap = classify_gap(s, j)
        print(f"{cat:<25} {s:>6} {j:>6} {ratio:>8}  {gap}")

        if "Well-tracked" in gap:
            well_tracked.append(cat)
        elif "Under-tracked" in gap:
            under_tracked.append(cat)
        elif "CRITICAL" in gap or "HIGH" in gap:
            invisible.append(cat)

    # Uncategorized
    s_unc = slack_counts.get("uncategorized", 0)
    j_unc = jira_counts.get("uncategorized", 0)
    if s_unc or j_unc:
        print(f"{'uncategorized':<25} {s_unc:>6} {j_unc:>6}")

    print()
    print(f"Total human Slack messages: {len(slack_msgs)}")
    print(f"Total SREIN tickets:        {len(jira_msgs)}")
    print(f"Ticket-to-Slack ratio:      {len(jira_msgs)/len(slack_msgs):.0%}" if slack_msgs else "N/A")

    print()
    print("TRACKING ASSESSMENT")
    print("-" * 50)
    if well_tracked:
        print(f"  Well-tracked:    {', '.join(well_tracked)}")
    if under_tracked:
        print(f"  Under-tracked:   {', '.join(under_tracked)}")
    if invisible:
        print(f"  Invisible work:  {', '.join(invisible)}")

    print()
    print("RECOMMENDATIONS")
    print("-" * 50)
    for cat in invisible:
        s = slack_counts.get(cat, 0)
        print(f"  [{cat}] {s} Slack messages, 0 tickets")
        print(f"    → Create tracking mechanism or self-service docs")
        print()


if __name__ == "__main__":
    main()
