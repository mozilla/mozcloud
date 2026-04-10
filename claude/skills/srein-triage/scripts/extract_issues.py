#!/usr/bin/env python3
"""Extract issue summaries from a saved JIRA JQL results JSON file.

Usage: python3 extract_issues.py <json_file>

Prints a structured summary of each issue: key, summary, reporter,
dates, priority, comment count/last date, linked issues, and
description excerpt.
"""
import json
import sys


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <json_file>", file=sys.stderr)
        sys.exit(1)

    with open(sys.argv[1]) as f:
        data = json.load(f)

    issues = data["issues"]
    print(f"Total issues: {issues['totalCount']}")

    for issue in issues["nodes"]:
        key = issue["key"]
        fields = issue["fields"]
        reporter = fields["reporter"]
        comments = fields["comment"]["comments"]
        links = fields["issuelinks"]
        desc = (fields.get("description") or "")[:500]
        created = fields["created"][:10]
        updated = fields["updated"][:10]
        priority = fields["priority"]["name"]
        last_comment = comments[-1]["created"][:10] if comments else "none"
        last_author = comments[-1]["author"]["displayName"] if comments else "n/a"

        print()
        print(f"=== {key}: {fields['summary']} ===")
        print(f"  Reporter: {reporter['displayName']} ({reporter['emailAddress']}) ID:{reporter['accountId']}")
        print(f"  Created: {created}, Updated: {updated}, Priority: {priority}")
        print(f"  Comments: {len(comments)}, Last: {last_comment} by {last_author}")
        print(f"  Links: {len(links)}")
        for link in links:
            if "outwardIssue" in link:
                li = link["outwardIssue"]
                print(f"    -> {li['key']}: {li['fields']['summary']}")
            if "inwardIssue" in link:
                li = link["inwardIssue"]
                print(f"    <- {li['key']}: {li['fields']['summary']}")
        print(f"  Desc: {desc[:300]}")


if __name__ == "__main__":
    main()
