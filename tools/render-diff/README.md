# Render-Diff
render-diff is intended to render helm charts stored in our infrastructure git repositories and compare the current revision vs a target revision (defaults to main)

# Setup
### Requirements
* `make`
* `git`
* Go `1.25` or newer

# Installation

`go install github.com/mozilla/mozcloud/tools/render-diff@latest`

# Flags

* `-chart-path` - Relative path to the chart (required).
* `-ref` - Target Git ref to compare against (e.g., 'test', 'develop') (default "main").
* `-values` - Path to an additional values file, relative to the chart-path (can be specified multiple times). The chart's `values.yaml` is always included first.

# Examples

### This should be run while your current directory is within your git repository

#### Checking diff against another target ref
* ```render-diff -chart-path=./cicd-demos/k8s/cicd-demos -values=values-dev.yaml --ref notifications-testing-2```
#### Checking diff against the main branch
* ```render-diff -chart-path=./cicd-demos/k8s/cicd-demos -values=values-dev.yaml```