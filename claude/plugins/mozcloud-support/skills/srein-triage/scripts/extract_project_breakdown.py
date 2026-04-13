#!/usr/bin/env python3
"""Extract project breakdown from a saved JIRA JQL results JSON file.

Usage: python3 extract_project_breakdown.py <json_file> [json_file ...]

For each file, prints total issue count and a project-key frequency
distribution. Useful for deriving reporter team affiliation from their
all-projects query results.
"""
import json
import sys


def process_file(path):
    with open(path) as f:
        data = json.load(f)

    issues = data["issues"]
    total = issues["totalCount"]

    projects = {}
    for issue in issues["nodes"]:
        key = issue["fields"]["project"]["key"]
        projects[key] = projects.get(key, 0) + 1

    print(f"=== {path} ===")
    print(f"Total: {total}")
    print(projects)


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <json_file> [json_file ...]", file=sys.stderr)
        sys.exit(1)

    for path in sys.argv[1:]:
        process_file(path)


if __name__ == "__main__":
    main()
