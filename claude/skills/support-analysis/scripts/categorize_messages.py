#!/usr/bin/env python3
"""
categorize_messages.py - Categorize support messages by theme using keyword matching.

Usage:
    cat messages.json | python3 categorize_messages.py
    python3 categorize_messages.py < messages.json
    python3 categorize_messages.py --file messages.json

Input: JSON array of objects with at least a "text" field.
       Optional fields: "user", "ts", "source" (slack|jira), "key" (jira ticket key)

Output: JSON array with added "categories" and "is_automated" fields, plus summary stats.

The input can come from Slack channel export or Jira ticket export.
"""

import json
import sys
import argparse
import re
from collections import Counter

# Category definitions with keyword patterns
CATEGORIES = {
    "access-permissions": [
        r"\baccess\b", r"\bgrant\b", r"\bpermission", r"\b404\b.*repo",
        r"can't see", r"can't access", r"\bviewer\b", r"\beditor\b",
        r"added to.*repo", r"added to.*project", r"\brole\b",
    ],
    "deployment-argocd": [
        r"\bargo\b", r"\bargocd\b", r"\bdeploy", r"image.*not.*pick",
        r"\bsync\b.*argo", r"\brollout", r"not deploying", r"\bstuck\b.*deploy",
        r"\bdegraded\b",
    ],
    "terraform-atlantis": [
        r"\batlantis\b", r"\bterraform\b", r"\bopentofu\b",
        r"atlantis.*plan", r"atlantis.*apply", r"\block\b.*atlantis",
        r"\bstate\b.*drift",
    ],
    "dns-networking": [
        r"\bcname\b", r"\bdomain\b", r"\bssl\b", r"\bcert\b",
        r"\bredirect\b", r"\bdns\b", r"route\s*53",
    ],
    "monitoring-grafana": [
        r"\bgrafana\b", r"\balert\b.*rule", r"\bdashboard\b",
        r"\bmetrics?\b", r"\btracing\b", r"\byardstick\b",
        r"\bprometheus\b", r"budget.*alert",
    ],
    "database": [
        r"\bdb\b", r"\bcloudsql\b", r"\bmysql\b", r"\bpostgres",
        r"\bconnection.*limit", r"disk.*size", r"\brds\b",
        r"database", r"\bredis\b",
    ],
    "gcp-config": [
        r"\bgcp\b.*project", r"\bbilling\b", r"\bbucket\b",
        r"\bvertex\b", r"\biam\b", r"service.*account",
        r"gcp.*access",
    ],
    "helm-k8s": [
        r"\bhelm\b", r"\bchart\b", r"\bdeployment\b.*k8s",
        r"\bhpa\b", r"\bingress\b", r"\bnginx\b", r"\bk8s\b",
        r"\bpod\b", r"\bcontainer\b", r"mozcloud-workload",
        r"\bkubectl\b",
    ],
    "incident": [
        r"\bincident\b", r"\bdown\b", r"\b503\b", r"\boutage\b",
        r"\burgent\b", r"production.*issue", r"\bbroken\b",
        r"\b500\b.*error",
    ],
    "onboarding": [
        r"new.*tenant", r"new.*service", r"\bonboard",
        r"first.*time", r"getting.*started",
    ],
    "pr-review": [
        r":review:", r"review.*please", r"\bPR\b.*review",
        r"\bapprove\b.*pr", r"\blgtm\b",
    ],
    "documentation": [
        r"\bdocs?\b", r"\bdocumentation\b", r"where.*do.*i.*find",
        r"how.*do.*i\b", r"is.*there.*a.*guide", r"\brunbook\b",
    ],
}

# Patterns indicating automated/bot messages
BOT_PATTERNS = [
    r"mozilla-hub\.atlassian\.net/browse/SREIN-",  # Jira bot (any format)
    r"\bcreated a (Task|Bug|Story|Epic|Sub-task)\b",  # Jira creation notifications
    r"SRE BOT",  # Triage bot
    r"triage engineer",  # Triage rotation
    r"has joined the channel",
    r"has left the channel",
    r"set the channel",
    r"was added to",
    r"pinned a message",
    r"This message was deleted",
    r"Grafana Alert",
    r"grafana_alert",
]

# Known bot user IDs in #mozcloud-support
BOT_USER_IDS = {
    "U8TT4054G",  # Jira bot
}


def is_automated(text, user=None):
    """Check if a message is from a bot or automated system."""
    if user and user in BOT_USER_IDS:
        return True
    for pattern in BOT_PATTERNS:
        if re.search(pattern, text, re.IGNORECASE):
            return True
    return False


def categorize(text):
    """Return list of matching categories for a message."""
    matches = []
    text_lower = text.lower()
    for category, patterns in CATEGORIES.items():
        for pattern in patterns:
            if re.search(pattern, text_lower):
                matches.append(category)
                break
    return matches if matches else ["uncategorized"]


def main():
    parser = argparse.ArgumentParser(description="Categorize support messages")
    parser.add_argument("--file", "-f", help="Input JSON file (default: stdin)")
    parser.add_argument("--summary-only", "-s", action="store_true",
                        help="Only print summary stats, not full output")
    args = parser.parse_args()

    if args.file:
        with open(args.file) as f:
            messages = json.load(f)
    else:
        messages = json.load(sys.stdin)

    category_counts = Counter()
    automated_count = 0
    human_count = 0
    results = []

    for msg in messages:
        text = msg.get("text", "") or msg.get("summary", "")
        user = msg.get("user")
        auto = is_automated(text, user)
        cats = categorize(text) if not auto else ["automated"]

        if auto:
            automated_count += 1
        else:
            human_count += 1
            for cat in cats:
                category_counts[cat] += 1

        result = {**msg, "categories": cats, "is_automated": auto}
        results.append(result)

    # Print summary
    print("=" * 60, file=sys.stderr)
    print("CATEGORIZATION SUMMARY", file=sys.stderr)
    print("=" * 60, file=sys.stderr)
    print(f"Total messages:    {len(messages)}", file=sys.stderr)
    print(f"Human messages:    {human_count}", file=sys.stderr)
    print(f"Automated:         {automated_count}", file=sys.stderr)
    print(file=sys.stderr)
    print("Categories (human messages only):", file=sys.stderr)
    print("-" * 40, file=sys.stderr)
    for cat, count in category_counts.most_common():
        print(f"  {cat:<25} {count:>4}", file=sys.stderr)
    print("=" * 60, file=sys.stderr)

    if not args.summary_only:
        json.dump(results, sys.stdout, indent=2)


if __name__ == "__main__":
    main()
