---
name: tenant-bootstrap
description: >
    Bootstrap a new MozCloud tenant step-by-step across the three sequential
    PRs: the GCP project entry in `<function>-infra`, the project/environment
    resources scaffold, and the tenant definition in `global-platform-admin`.
    Pauses between PRs for merge confirmation. Only trigger on explicit
    /tenant-bootstrap invocations.
user-invocable: true
disable-model-invocation: true
allowed-tools:
  - AskUserQuestion
  - Bash(git clone:*)
  - Bash(git checkout:*)
  - Bash(git add:*)
  - Bash(git commit:*)
  - Bash(git push:*)
  - Bash(./misc/bin/new-project:*)
  - Bash(mkdir:*)
  - Bash(ls:*)
  - Read
  - Write
  - Edit
---

# Bootstrap a new MozCloud tenant

You are guiding the user through the full MozCloud tenant bootstrap process, covering pre-flight input collection, repo checkout, and three sequential PRs. Follow this workflow exactly.

Reference doc: https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/2408480771/Bootstrapping+a+tenant

---

## Phase 1: Collect inputs

Ask in three batches.

### Batch 1 of 3

Ask all four questions at once:

1. **app_code** — The tenant identifier. Must be 15 characters or fewer (GCP project ID limit). Ask the user to enter it.
2. **function** — Which infra repo this tenant belongs in. Options: `webservices` (default, for most services), `dataservices` (GCP data products: BigQuery, Dataflow, etc.), `sandbox` (testing/experimental).
3. **risk_level** — Options: `high` (default, use unless you have a specific reason not to) or `low`.
4. **workgroup** — The workgroup that owns this tenant. In most cases this matches the app_code. Ask: "Does your workgroup name match the app_code? If not, provide the workgroup name."

### Batch 2 of 3

Ask all four questions at once:

1. **application_repository** — The GitHub repo containing the application source code. Format: `mozilla/<repo-name>` (e.g. `mozilla/my-service`).
2. **image_name** — The container image name as it will appear in GAR (e.g. `my-service`).
3. **application_ports** — Comma-separated list of ports the application listens on. Default is `8080`. Controls Kubernetes NetworkPolicy ingress rules.
4. **workgroup_status** — Does the workgroup already exist? Options: "Yes, it exists" or "No, I need to create one".

### Batch 3 of 3

Ask all four questions at once. The last three are optional:

1. **clone_base_path** — Where should the repos be cloned? (e.g. `~/src/mozilla` or `/tmp/bootstrap`). Used for any repo the user does not provide an explicit path for below. The repos will be cloned as subdirectories of this path.
2. **infra_repo_path** (optional) — Path to an existing `<function>-infra` checkout to reuse (e.g. `~/work/webservices-infra`). Leave blank to clone under `clone_base_path`.
3. **gpa_repo_path** (optional) — Path to an existing `global-platform-admin` checkout to reuse. Leave blank to clone under `clone_base_path`.
4. **jira_ticket** (optional) — Tracking ticket for this bootstrap (e.g. `MZCLD-1234`). If provided, it will be included in the suggested PR bodies so the target repos' autolink configuration renders it as a hyperlink. Leave blank if there is no tracking ticket.

---

## Phase 2: Validate inputs and pre-flight checks

After collecting all inputs, perform these checks before doing anything:

### Validate app_code length

If `app_code` is longer than 15 characters:
- Warn the user: "Your app_code is N characters long. GCP project IDs are capped at 30 characters, and the `moz-fx-` + `-nonprod` prefix/suffix uses 15 characters. You will need to add `random_id = true` to your locals.tf entry."
- Continue — do not halt; just flag this so the locals.tf edit is correct.

### Check workgroup status

If workgroup does not exist:
- Tell the user: "You must create a workgroup before proceeding. Check https://protosaur.dev/dawg/ to verify no existing workgroup fits. Then follow https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/2492956683 and file an SREIN ticket at https://mozilla-hub.atlassian.net/jira/software/c/projects/SREIN/form/1344. Workgroup tickets typically resolve within a few business days. Return to this workflow once the ticket is resolved."
- **Halt here.** Do not proceed until the user confirms the workgroup exists.

### Derive values

Compute the working paths:
- `infra_path` = `infra_repo_path` if provided, else `<clone_base_path>/<function>-infra`
- `gpa_path` = `gpa_repo_path` if provided, else `<clone_base_path>/global-platform-admin`

Display these for the user to confirm before proceeding:

```
infra_repo:          mozilla/<function>-infra
infra_path:          <infra_path>
gpa_path:            <gpa_path>
nonprod_project:     moz-fx-<app_code>-nonprod
prod_project:        moz-fx-<app_code>-prod
image_repository:    us-docker.pkg.dev/moz-fx-<app_code>-prod/<app_code>-prod/<image_name>
target_cluster:      <function>-<risk_level>
workgroup:           <workgroup>
```

Ask the user to confirm these look correct before continuing.

---

## Phase 3: Clone or reuse repos

For each of the two repos (`<function>-infra` at `<infra_path>`, `global-platform-admin` at `<gpa_path>`):

- If the user provided an explicit path (`infra_repo_path` / `gpa_repo_path`) or a checkout already exists at the derived path, reuse it — do a `git checkout main && git pull` in that directory to make sure it's current.
- Otherwise, clone it:

```bash
git clone git@github.com:mozilla/<function>-infra.git <infra_path>
git clone git@github.com:mozilla/global-platform-admin.git <gpa_path>
```

---

## Phase 4: Step 1 — PR 1 (GCP project entry)

**Goal:** Add the GCP project entry to `projects/tf/global/locals.tf` in `<function>-infra`.

1. Create a branch:

```bash
cd <infra_path>
git checkout -b bootstrap-<app_code>-pr1
```

2. Edit `projects/tf/global/locals.tf`. Find the `locals { projects = {` block and add an entry following the existing pattern. The entry format is:

```hcl
<app_code> = {}
```

If `app_code` is longer than 15 characters, use:

```hcl
<app_code> = { random_id = true }
```

Use the Read tool to read the current file first and find the right insertion point. Follow the existing formatting exactly.

3. Commit and push:

```bash
cd <infra_path>
git add projects/tf/global/locals.tf
git commit -m "bootstrap: add <app_code> GCP project entry"
git push -u origin bootstrap-<app_code>-pr1
```

4. Print instructions for the user:

```
PR 1 is ready to open. Steps:
1. Open a PR from branch bootstrap-<app_code>-pr1 against main in mozilla/<function>-infra
2. Atlantis will automatically run a plan when the PR is opened
3. Once a code owner approves, comment: atlantis apply
4. If the first apply fails with an unexpected error, comment atlantis apply a second time (known issue MZCLD-787)
5. Fix any other failures before retrying
6. MERGE ONLY after the apply succeeds
7. Do NOT start Step 2 until this PR is fully merged
```

Then print the suggested PR body. Omit the `<jira_ticket>` bullet if the user did not provide a ticket.

```
Suggested PR body:

## Description

Bootstrap PR 1/3 for tenant <app_code>. Adds the GCP project entry to `projects/tf/global/locals.tf`.

## Related Tickets & Documents

* <jira_ticket>
* https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/2408480771/Bootstrapping+a+tenant
```

5. **Pause.** Use AskUserQuestion to ask: "Has PR 1 been merged and the apply succeeded? Reply yes to continue to Step 2."

---

## Phase 5: Step 2 — PR 2 (project and environment resources)

**Goal:** Run the scaffold script, update permissions if needed, add CODEOWNERS entry.

1. Create a branch:

```bash
cd <infra_path>
git checkout main && git pull
git checkout -b bootstrap-<app_code>-pr2
```

2. Run the scaffold script:

```bash
cd <infra_path>
./misc/bin/new-project <app_code> --risk <risk_level>
```

3. If `workgroup` differs from `app_code`:
   - Read `<app_code>/tf/global/permissions.tf`
   - Replace all three `workgroup:<app_code>/` references with `workgroup:<workgroup>/`
   - The three references are the values for `admin_ids`, `developer_ids`, and `viewer_ids` in the permissions module

4. Add CODEOWNERS entry. Read the CODEOWNERS file, find the end of the file (or the section for app directories), and append:

```
/<app_code>/  @mozilla/<workgroup>-workgroup
```

Follow the existing formatting in the file.

5. Commit and push:

```bash
cd <infra_path>
git add <app_code>/ CODEOWNERS
git commit -m "bootstrap: add <app_code> project and environment resources"
git push -u origin bootstrap-<app_code>-pr2
```

6. Print instructions:

```
PR 2 is ready to open. Steps:
1. Open a PR from branch bootstrap-<app_code>-pr2 against main in mozilla/<function>-infra
2. Atlantis will automatically run a plan when the PR is opened
3. Once a code owner approves, comment: atlantis apply
4. Fix any failures and re-run if needed
5. MERGE ONLY after the apply succeeds
6. Do NOT start Step 3 until this PR is fully merged

Note: Do not add custom Terraform (Redis, CloudSQL, etc.) to this PR.
Custom resources may fail until the shared VPC is provisioned in Step 3.
You will not have a Kubernetes namespace until Step 3 is complete.
```

Then print the suggested PR body. Omit the `<jira_ticket>` bullet if the user did not provide a ticket.

```
Suggested PR body:

## Description

Bootstrap PR 2/3 for tenant <app_code>. Adds project and environment resources via the `new-project` scaffold, plus a CODEOWNERS entry.

## Related Tickets & Documents

* <jira_ticket>
* https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/2408480771/Bootstrapping+a+tenant
```

7. **Pause.** Use AskUserQuestion to ask: "Has PR 2 been merged and the apply succeeded? Reply yes to continue to Step 3."

---

## Phase 6: Step 3 — PR 3 (tenant definition)

**Goal:** Create the tenant YAML in `global-platform-admin`.

1. Create a branch:

```bash
cd <gpa_path>
git checkout main && git pull
git checkout -b bootstrap-<app_code>-pr3
```

2. Parse the `application_ports` input into a YAML list. For example, `8080, 9090` becomes:

```yaml
    - 8080
    - 9090
```

3. Write the file `<gpa_path>/tenants/<app_code>.yaml` using the template at [templates/tenant.yaml](templates/tenant.yaml). Substitute every `<placeholder>` with the collected/derived input values. The `application_ports` list goes where `<application_ports_as_yaml_list>` appears, indented to match.

4. Commit and push:

```bash
cd <gpa_path>
git add tenants/<app_code>.yaml
git commit -m "bootstrap: add <app_code> tenant definition"
git push -u origin bootstrap-<app_code>-pr3
```

5. Print the PR instructions from [references/pr3-atlantis-sequence.md](references/pr3-atlantis-sequence.md). Substitute `<function>` and `<risk_level>` with the collected values before printing.

---

## Phase 7: Verification

Once PR 3 is merged, print the verification checklist from [references/post-merge-verification.md](references/post-merge-verification.md). Substitute `<app_code>` with the collected value and pick the correct ArgoCD URL for the `<function>` the user chose (the reference file lists all three).
