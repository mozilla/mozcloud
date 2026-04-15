package gsm

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"charm.land/huh/v2"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/gsm"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
)

// ErrAborted is returned when the user cancels an interactive operation.
var ErrAborted = errors.New("aborted")

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit a secret in $EDITOR",
	Long: `Fetch a secret, open it in your editor, validate JSON, and push a new version if changed.

Accepts input on stdin for non-interactive use:

  echo '{"key":"value"}' | mzcld gsm edit -p my-project -s my-secret
  cat config.json | mzcld gsm edit -p my-project -s my-secret`,
	RunE:  runEdit,
}

func init() {
	editCmd.Flags().StringP("project", "p", "", "GCP project ID")
	editCmd.Flags().StringP("secret", "s", "", "Secret name")
	editCmd.Flags().StringP("version", "v", "latest", "Version to edit (default: latest)")
	editCmd.Flags().Bool("create", false, "Allow creating a new secret")
}

func runEdit(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	flagProject, _ := cmd.Flags().GetString("project")
	flagSecret, _ := cmd.Flags().GetString("secret")
	flagVersion, _ := cmd.Flags().GetString("version")

	// Check if stdin has data piped in and read it immediately,
	// blocking until the upstream process finishes.
	stdinPiped := false
	var stdinData []byte
	if fi, _ := os.Stdin.Stat(); fi != nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		stdinPiped = true
		var err error
		stdinData, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
	}

	projectID, err := selectProject(ctx, flagProject)
	if err != nil {
		return err
	}

	client, err := gsm.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close() //nolint:errcheck

	allowCreate, _ := cmd.Flags().GetBool("create")
	secretName, isNew, err := selectSecret(ctx, client, projectID, flagSecret, allowCreate)
	if err != nil {
		return err
	}

	// Fetch current content or start with empty JSON.
	var content []byte
	if isNew {
		content = []byte("{}\n")
	} else {
		runWithSpinner(ctx, "Fetching secret...", func() {
			content, err = client.GetSecretVersion(ctx, projectID, secretName, flagVersion)
		})
		if err != nil {
			return err
		}
	}

	originalSum := sha256sum(content)

	var newContent []byte
	if stdinPiped {
		newContent = stdinData
	} else {
		// Interactive: write to temp file and open editor.
		tmp, err := os.CreateTemp("", "mzcld-gsm-*.json")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmp.Name()) //nolint:errcheck

		// Go 1.16+ creates temp files with 0600 by default.
		if _, err := tmp.Write(content); err != nil {
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		tmp.Close() //nolint:errcheck

		// Detect if content is JSON for validation.
		isJSON := json.Valid(content)

		if err := editFileLoop(tmp.Name(), isJSON); err != nil {
			return err
		}

		newContent, err = os.ReadFile(tmp.Name())
		if err != nil {
			return fmt.Errorf("failed to read edited file: %w", err)
		}
	}

	newSum := sha256sum(newContent)
	if originalSum == newSum {
		ui.Info("No changes. Not pushing new version.")
		return nil
	}

	// Create the secret if it's new, then push the version.
	runWithSpinner(ctx, "Pushing new version...", func() {
		if isNew {
			err = client.CreateSecret(ctx, projectID, secretName)
			if err != nil {
				return
			}
		}
		err = client.AddSecretVersion(ctx, projectID, secretName, newContent)
	})
	if err != nil {
		return err
	}

	cacheSelection(projectID, secretName)
	ui.Success(fmt.Sprintf("New version of %s pushed.", secretName))
	return nil
}

// editFileLoop opens the file in $EDITOR. If validateJSON is true, it validates
// the content as JSON after each edit and loops on failure.
func editFileLoop(path string, validateJSON bool) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	for {
		c := exec.Command(editor, path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		if !validateJSON {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		if json.Valid(data) {
			return nil
		}

		ui.Warn("Invalid JSON.")
		var tryAgain bool
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Try again?").
					Value(&tryAgain),
			),
		).Run(); err != nil {
			return err
		}
		if !tryAgain {
			return ErrAborted
		}
	}
}

func sha256sum(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}
