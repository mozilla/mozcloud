# Render-Diff
`render-diff` provides a fast and local preview of rendered Kubernetes manifest changes.

It compares your locally rendered Helm chart or Kustomize overlay against the rendered output from a target Git ref (for example, `main` or `develop`), and prints a colored diff of the resulting YAML manifests.

It will default to a simple text based diff, but can use the [dyff](https://github.com/homeport/dyff) rendering engine with the `--semantic` flag.

## Requirements
* `make`
* `git`
* Go `1.24` or newer

## Installation

You can install `render-diff` directly using `go install`:

```sh
go install github.com/mozilla/mozcloud/tools/render-diff@latest
```

## Flags

| Flag | Shorthand | Description | Default |
| :--- | :--- | :--- | :--- |
| `--path` | `-p` | **(Required)** Relative path to the chart or kustomization directory. | `.` |
| `--ref` | `-r` | Target Git ref to compare against. | `main` |
| `--values` | `-f` | "Path to an additional values file (can be specified multiple times). The chart's default values.yaml is always loaded first" | `[]` |
| `--release-name` | | "Helm release name to use when rendering templates. Defaults to chart name" | `""` |
| `--update` | `-u` | Update helm chart dependencies. Required if lockfile does not match dependencies | `false` |
| `--semantic` | `-s` |  Enable semantic diffing of k8s manifests (using dyff) | `false` |
| `--debug` | `-d` | Enable verbose logging for debugging | `false` |
| `--version` | | Prints the application version. | |
| `--help` | `-h` | Show help information. | |

## Examples

Run this tool from within your Git repository. For Helm charts, values.yaml is automatically included.


#### Checking a Helm Chart diff against another target ref
* ```render-diff --path ./examples/helm/helloWorld --values values-dev.yaml --ref development```
#### Checking Kustomize diff against the default (`main`) branch
* ```render-diff -p ./examples/kustomize/helloWorld```
#### Checking Kustomize diff against a tag
* ```render-diff -p ./examples/kustomize/helloWorld -r tags/v0.5.1```
