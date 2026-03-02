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

✓ **GOOD Examples:**
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

✗ **BAD Examples (relative paths can fail if working directory changes):**
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

## Summary

1. **Capture `$CHART_DIR` at the start** of every migration session
2. **Use absolute paths** with `$CHART_DIR` for all file operations
3. **Verify location** after helm commands that might change directory
4. **Sanity check** before creating migration artifacts
