# Render-Diff
`render-diff` provides a fast and local preview of your rendered Kubernetes manifest changes.

It renders your local Helm chart or Kustomize overlay to compare the resulting manifests against the version in a target git ref (like 'main' or 'develop').
It prints a colored diff of the final rendered YAML.

## Requirements
* `make`
* `git`
* Go `1.24` or newer

## Installation

You can install `render-diff` directly using `go install`:

```sh
go install github.com/mozilla/mozcloud/tools/render-diff@latest
```

# Flags

| Flag | Shorthand | Description | Default |
| :--- | :--- | :--- | :--- |
| `--path` | `-p` | **(Required)** Relative path to the chart or kustomization directory. | `.` |
| `--ref` | `-r` | Target Git ref to compare against. | `main` |
| `--values` | `-f` | "Path to an additional values file (can be specified multiple times). The chart's default values.yaml is always loaded first" | `[]` |
| `--debug` | `-d` | Enable verbose logging for debugging | `false` |
| `--version` | | Prints the application version. | |
| `--help` | `-h` | Show help information. | |

# Examples

### This must be run while your current directory is within your git repository

#### Checking a Helm Chart diff against another target ref
* ```render-diff -path ./examples/helm/helloWorld -values values-dev.yaml --ref development```
#### Checking Kustomize diff against the default (`main`) branch
* ```render-diff -path ./examples/kustomize/helloWorld```
#### Checking Kustomize diff against a tag
* ```render-diff -p ./examples/kustomize/helloWorld -r tags/v0.5.1```
