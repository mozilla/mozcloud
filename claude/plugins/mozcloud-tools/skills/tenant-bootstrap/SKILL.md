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
  - Bash(tflint:*)
  - Bash(mkdir:*)
  - Bash(ls:*)
  - Read
  - Write
  - Edit
---

# Bootstrap a new MozCloud tenant

You are guiding the user through the full MozCloud tenant bootstrap process: pre-flight input collection, repo checkout, three sequential PRs (`<function>-infra` project entry, `<function>-infra` scaffold, `global-platform-admin` tenant definition), and a final no-op PR against `<function>-infra` to re-run `atlantis apply` and converge infra state. Follow this workflow exactly.

Reference doc: https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/2408480771/Bootstrapping+a+tenant

---

## Phase 1: Collect inputs

Ask in three batches.

### Batch 1 of 3

Ask all five questions at once:

1. **app_code** — The tenant identifier. Must be 15 characters or fewer (GCP project ID limit). Ask the user to enter it.
2. **function** — Which infra repo this tenant belongs in. Options: `webservices` (default, for most services), `dataservices` (GCP data products: BigQuery, Dataflow, etc.), `sandbox` (testing/experimental).
3. **risk_level** — Options: `high` (default, use unless you have a specific reason not to) or `low`.
4. **workgroup** — The workgroup that owns this tenant. In most cases this matches the app_code. Ask: "Does your workgroup name match the app_code? If not, provide the workgroup name."
5. **workgroup_status** — Does the workgroup already exist? Options: "Yes, it exists" or "No, I need to create one".

### Batch 2 of 3

Ask all three questions at once:

1. **application_repository** — The GitHub repo containing the application source code. Format: `mozilla/<repo-name>` (e.g. `mozilla/my-service`).
2. **image_name** — The container image name as it will appear in GAR (e.g. `my-service`).
3. **application_ports** — Comma-separated list of ports the application listens on. Default is `8080`. Controls Kubernetes NetworkPolicy ingress rules.

### Batch 3 of 3

Ask all three questions at once. The last is optional:

1. **infra_path** — Path to the `<function>-infra` checkout (e.g. `~/src/mozilla/webservices-infra`). If a checkout already exists at this path we'll reuse it; otherwise we'll clone it there.
2. **gpa_path** — Path to the `global-platform-admin` checkout (e.g. `~/src/mozilla/global-platform-admin`). Same behavior: reuse if it exists, otherwise clone there.
3. **jira_ticket** (optional) — Tracking ticket for this bootstrap (e.g. `MZCLD-1234`). If provided, it will be included in the suggested PR bodies so the target repos' autolink configuration renders it as a hyperlink. Leave blank if there is no tracking ticket.

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

Display the following for the user to confirm before proceeding:

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

Handle each repo independently.

**For `<function>-infra` at `<infra_path>`:**

- If a checkout already exists at `<infra_path>`, reuse it: run `git checkout main && git pull` in that directory to make sure it's current.
- Otherwise, clone it:

```bash
git clone git@github.com:mozilla/<function>-infra.git <infra_path>
```

**For `global-platform-admin` at `<gpa_path>`:**

- If a checkout already exists at `<gpa_path>`, reuse it: run `git checkout main && git pull` in that directory to make sure it's current.
- Otherwise, clone it:

```bash
git clone git@github.com:mozilla/global-platform-admin.git <gpa_path>
```

---

## Phase 4: Step 1 — PR 1 (GCP project entry)

**Goal:** Add the GCP project entry to `projects/tf/global/locals.tf` in `<function>-infra`.

1. Create a branch:

```bash
cd <infra_path>
git checkout main && git pull
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

3. Commit:

```bash
cd <infra_path>
git add projects/tf/global/locals.tf
git commit -m "feat(<app_code>): add GCP project entry"
```

Then run `git show HEAD --stat` so the user can review what was committed, and print:

```
Ready to push. The following command will run only after you confirm:

  git push -u origin bootstrap-<app_code>-pr1

Reply "yes" to push, or tell me what to change.
```

4. Push after the user replies "yes":

```bash
cd <infra_path>
git push -u origin bootstrap-<app_code>-pr1
```

5. Print instructions for the user:

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

6. **Pause.** Print the message below and **end your turn** — do not call any tool. Getting to a merged apply can take a long time, so this is a genuine hand-off, not a quick approval. When the user replies, parse the response for a confirmation and an optional PR URL; store the URL as `pr1_url` if provided, and proceed to Phase 5.

```
PR 1 is ready for review and merge. Once it has been merged and the atlantis
apply has succeeded, reply here with "yes" to continue to Step 2.
If you have a PR URL handy, paste it in the same reply and I'll
reference it from the PR 2 and PR 3 bodies (otherwise just reply "yes").
```

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

3. Run tflint against the generated tree to clean up unused declarations and surface any other issues:

```bash
cd <infra_path>/<app_code>
tflint --recursive --fix
```

Then read the output and handle it as follows:

- **`terraform_unused_declarations` warnings** are auto-fixed by `--fix`. These are expected for bootstraps that don't set up Cloudarmor (the scaffold emits `shared_infra_project_id_{prod,nonprod}` locals that only Cloudarmor tenants consume). Note in your summary that they were fixed.
- **Any other warning or error** → do **not** auto-fix. Print the full tflint output to the user, briefly describe each finding, ask how to proceed, and wait for direction before continuing.

If `tflint` is not installed (command not found), do **not** halt. Tell the user: "tflint is not installed locally, so I'm skipping the pre-commit lint; CI will run the same checks on the PR. For a fresh bootstrap, expect `terraform_unused_declarations` warnings on `shared_infra_project_id_prod` and `shared_infra_project_id_nonprod` — these are emitted by the scaffold for tenants that don't use Cloudarmor and can be ignored. Any other tflint finding should be fixed before merging." Then continue to the next step.

4. If `workgroup` differs from `app_code`:
   - Read `<app_code>/tf/global/permissions.tf`
   - Replace all three `workgroup:<app_code>/` references with `workgroup:<workgroup>/`
   - The three references are the values for `admin_ids`, `developer_ids`, and `viewer_ids` in the permissions module

5. Add CODEOWNERS entry. Read the CODEOWNERS file, find the block of `<app_code>/` entries, and insert the new line in its correct alphabetical position relative to its neighbors. Do **not** reorder any existing lines. The new line format is:

```
<app_code>/ @mozilla/<workgroup>-wg @mozilla/sre-wg
```

Note: no leading slash, `-wg` suffix (not `-workgroup`), and `@mozilla/sre-wg` as the co-owner. Follow the existing formatting in the file for spacing between the path and the owners.

6. Commit:

```bash
cd <infra_path>
git add <app_code>/ CODEOWNERS
git commit -m "feat(<app_code>): add project and environment resources"
```

Then run `git show HEAD --stat` so the user can review what was committed, and print:

```
Ready to push. The following command will run only after you confirm:

  git push -u origin bootstrap-<app_code>-pr2

Reply "yes" to push, or tell me what to change.
```

7. Push after the user replies "yes":

```bash
cd <infra_path>
git push -u origin bootstrap-<app_code>-pr2
```

8. Print instructions:

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

Then print the suggested PR body. Omit any bullet whose value was not provided (`<jira_ticket>`, `<pr1_url>`).

```
Suggested PR body:

## Description

Bootstrap PR 2/3 for tenant <app_code>. Adds project and environment resources via the `new-project` scaffold, plus a CODEOWNERS entry.

## Related Tickets & Documents

* <jira_ticket>
* PR 1: <pr1_url>
* https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/2408480771/Bootstrapping+a+tenant
```

9. **Pause.** Print the message below and **end your turn** — do not call any tool. Getting to a merged apply can take a long time, so this is a genuine hand-off, not a quick approval. When the user replies, parse the response for a confirmation and an optional PR URL; store the URL as `pr2_url` if provided, and proceed to Phase 6.

```
PR 2 is ready for review and merge. Once it has been merged and the atlantis
apply has succeeded, reply here with "yes" to continue to Step 3.
If you have a PR URL handy, paste it in the same reply and I'll
reference it from the PR 3 body (otherwise just reply "yes").
```

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

After writing the file, tell the user:

- The template's Image Updater defaults assume `latest`-tagged builds for dev and semver-tagged builds for stage/prod (e.g. `1.2.3`). If the tenant uses a `v` prefix, adjust the regex to `^v?\d+\.\d+\.\d+$`; for other tagging schemes, see existing tenants in `global-platform-admin/tenants/` for examples and `tenants/schema.json` for the field reference.
- Argo auto-sync is **not** set in the tenant YAML — it's configured on the generated ArgoCD Applications. After PR 3 merges, we suggest enabling auto-sync for the `stage` Application and leaving it disabled for `prod` (this will also appear in the post-bootstrap report).

4. Commit:

```bash
cd <gpa_path>
git add tenants/<app_code>.yaml
git commit -m "feat(<app_code>): add tenant definition"
```

Then run `git show HEAD --stat` so the user can review what was committed, and print:

```
Ready to push. The following command will run only after you confirm:

  git push -u origin bootstrap-<app_code>-pr3

Reply "yes" to push, or tell me what to change.
```

5. Push after the user replies "yes":

```bash
cd <gpa_path>
git push -u origin bootstrap-<app_code>-pr3
```

6. Print the PR instructions from [references/pr3-atlantis-sequence.md](references/pr3-atlantis-sequence.md). Substitute `<function>` and `<risk_level>` with the collected values before printing.

7. **Pause.** Print the message below and **end your turn** — do not call any tool. When the user replies, parse the response for a confirmation and an optional PR URL; store the URL as `pr3_url` if provided, and proceed to Phase 7.

```
PR 3 is ready for review and merge. Once it has been merged, reply here
with "yes" to continue to the final infra apply — bootstrap isn't
complete until that's done. If you have a PR URL handy, paste it in
the same reply and I'll reference it from the post-bootstrap report
(otherwise just reply "yes").
```

---

## Phase 7: Final infra apply

**Goal:** Run one more `atlantis apply` against `<function>-infra` to pick up the DNS records and logging-IAM bindings that depend on the tenant namespace created by PR 3. PR 2's apply ran before PR 3 existed, so those resources fell through to their `try()` fallbacks on the first pass.

1. **Pause.** Print the message below and **end your turn** — do not call any tool. When the user replies "yes", proceed to Phase 8.

```
One more apply to go. PR 2's Atlantis run happened before PR 3 existed,
so a few resources that depend on your tenant's namespace — the DNS
records and the logging-IAM bindings (logging_dataset_writer,
logging_bucket_writer) — fell through to their try() fallbacks and
weren't actually created. A second apply on <function>-infra picks
them up:

1. Open a no-op PR against mozilla/<function>-infra — a trivial change
   under <app_code>/tf/ (a comment edit in a *.tf file is fine) is
   enough to get Atlantis to plan.
2. Once Atlantis plans cleanly (the plan should include the DNS records
   and IAM bindings), comment: atlantis apply
3. Wait for the apply to succeed. You can close the PR without merging
   if it only existed to trigger this apply.

Reply "yes" once the apply has succeeded to render the post-bootstrap report.
```

---

## Phase 8: Post-bootstrap report

**Goal:** Render a report summarizing the bootstrap — PRs landed, verification steps the user should run, and what to do next. Print it in-chat; offer to save it to disk.

1. Build the report by substituting every `<placeholder>` in the template at [templates/post-bootstrap-report.md](templates/post-bootstrap-report.md) with the collected values. Omit the `**Tracking:**` line if no `jira_ticket` was provided. Omit any PR URL bullet whose corresponding `prN_url` was not provided.

2. Print the rendered report in a fenced markdown block so the user can copy it.

3. Ask the user whether they want a copy written to disk. End your turn with this message:

```
Would you like me to save this report to a file? If yes, reply with a
path (absolute or relative to the current directory). If no, just
reply "no" — you can copy the report above from this chat.
```

4. On the user's next reply:
   - If the reply looks like a filesystem path, use the Write tool to write the rendered report to that path, then print the absolute path so the user can locate it.
   - Otherwise, print a brief "Done. You can copy the report above." acknowledgement and end.
