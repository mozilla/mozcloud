# PR 3 Atlantis sequence

Print the block below to the user after pushing PR 3. Substitute `<function>` and `<risk_level>` with the collected values.

```
PR 3 is ready to open. Steps:
1. Open a PR from branch bootstrap-<app_code>-pr3 against main in mozilla/global-platform-admin
2. The PR requires approval from @mozilla/sre-wg (CODEOWNERS for this repo)
3. Autoplan does NOT run in this repo — you must trigger plan manually

Once approved, comment the following IN ORDER (run plan before each apply).
Review plan output before applying. If you see unexpected drift, ask in #mozcloud-support first.

# Provision shared platform resources for the new tenant
atlantis plan -d platform-shared/tf/global
atlantis apply -d platform-shared/tf/global

# Provision cluster-level resources (namespaces, quotas) in nonprod and prod
atlantis plan -d <function>-<risk_level>/tf/nonprod
atlantis apply -d <function>-<risk_level>/tf/nonprod
atlantis plan -d <function>-<risk_level>/tf/prod
atlantis apply -d <function>-<risk_level>/tf/prod

# Update VPC peering and firewall rules
atlantis plan -d functional-org-vpc/tf
atlantis apply -d functional-org-vpc/tf

# Update bastion host access rules
atlantis plan -d bastions/tf
atlantis apply -d bastions/tf

# Sync Wiz security tool user assignments (expect many wiz_user changes, safe to apply)
atlantis plan -d wiz/tf
atlantis apply -d wiz/tf

Fix any failures before continuing to the next directory.
MERGE ONLY after all applies succeed.

After merging: set up bastion access if you haven't already.
See https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/27919459
```
