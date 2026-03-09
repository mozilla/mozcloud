# Working Directory Management

**CRITICAL**: Maintain consistent working directory throughout the migration to avoid creating files in wrong locations.

## Capture Chart Root Directory

At the start of migration, capture and verify the chart root directory:

```bash
CHART_DIR=$(pwd)
echo "Chart directory: $CHART_DIR"

# Verify Chart.yaml exists
if [ ! -f "$CHART_DIR/Chart.yaml" ]; then
  echo "ERROR: Chart.yaml not found in $CHART_DIR"
  exit 1
fi
```

## Always Use Absolute Paths

**For ALL file operations, use the $CHART_DIR variable:**

[OK] **GOOD Examples:**
```bash
# Creating directories
mkdir -p $CHART_DIR/.migration/manifests/dev

# Rendering manifests
helm template . -f $CHART_DIR/values.yaml -f $CHART_DIR/values-dev.yaml > $CHART_DIR/.migration/manifests/dev/original.yaml

# Writing documentation
cat > $CHART_DIR/.migration/README.md <<EOF
...
EOF

# Reading files
cat $CHART_DIR/values.yaml
```

[FAIL] **BAD Examples (relative paths can fail if working directory changes):**
```bash
mkdir -p .migration/manifests/dev
helm template . -f values.yaml > .migration/manifests/dev/original.yaml
```

## Verify Working Directory After Helm Operations

Some helm operations may change the working directory context. Always verify location after:

```bash
# After helm pull
helm pull oci://... --untar --untardir $CHART_DIR/.migration/
cd $CHART_DIR  # Return to chart root
pwd            # Verify we're in the correct location

# After any helm command that might change context
cd $CHART_DIR
```

## Directory Creation Safety Check

Before creating any files in `.migration/`, verify location:

```bash
# Sanity check before file operations
if [ ! -f "$CHART_DIR/Chart.yaml" ]; then
  echo "ERROR: Not in chart directory. Current: $(pwd)"
  exit 1
fi

# Now safe to create files
mkdir -p $CHART_DIR/.migration/manifests/dev
```

**Why this matters:** Some helm operations (particularly `helm pull`) may change the working directory context. Using absolute paths with the `$CHART_DIR` variable prevents accidentally creating migration artifacts in nested or incorrect locations.

## File Write Restrictions

**CRITICAL SECURITY REQUIREMENT**: All file writes must be scoped to the current chart directory.

### Allowed Write Locations
- Chart configuration: `$CHART_DIR/values.yaml`, `$CHART_DIR/values-*.yaml`
- Chart definition: `$CHART_DIR/Chart.yaml`, `$CHART_DIR/Chart.lock`
- Templates: `$CHART_DIR/templates/*`
- Helpers: `$CHART_DIR/templates/_helpers.tpl`
- Migration artifacts: `$CHART_DIR/.migration/*`

### Prohibited Write Locations
- Parent directories: `../anything`
- Other tenant charts: `/path/to/other-chart/`
- System directories: `/tmp/`, `/var/`, `/etc/`, `/home/`
- Absolute paths outside chart: `/Users/`, `/opt/`, etc.
- Root directory: Any path not starting with `$CHART_DIR`

### Validation Before Writing

Always validate write path before creating files:

```bash
# Validate path is within chart directory
TARGET_PATH="/proposed/path/to/file"
if [[ "$TARGET_PATH" != "$CHART_DIR"* ]]; then
  echo "ERROR: Cannot write outside chart directory"
  echo "Target: $TARGET_PATH"
  echo "Chart dir: $CHART_DIR"
  exit 1
fi
```

### Handling User Requests to Write Outside Chart

If a user requests writing files outside the current chart directory:

1. **Refuse politely**: Explain the scope restriction
2. **Explain why**: Security and scope - skill is for migrating this chart only
3. **Suggest alternatives**: If they need to affect other charts, run the skill from that chart's directory

**Example Response:**
```
I cannot write to that location as it's outside the current chart directory.
This skill is scoped to migrate only the chart in the current directory
($CHART_DIR) to prevent accidental modifications to other tenants or system files.

If you need to migrate another chart, please run the skill from that chart's
directory instead.
```

## Summary

1. **Capture `$CHART_DIR` at the start** of every migration session
2. **Use absolute paths** with `$CHART_DIR` for all file operations
3. **Validate write paths** are within `$CHART_DIR` before creating files
4. **Refuse writes outside chart directory** and explain scope restriction
5. **Verify location** after helm commands that might change directory
6. **Sanity check** before creating migration artifacts
