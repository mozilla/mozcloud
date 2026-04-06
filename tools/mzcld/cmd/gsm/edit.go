package gsm

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/huh"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/gsm"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit a secret in $EDITOR",
	Long:  "Fetch a secret, open it in your editor, validate JSON, and push a new version if changed.",
	RunE:  runEdit,
}

func init() {
	editCmd.Flags().StringP("project", "p", "", "GCP project ID")
	editCmd.Flags().StringP("secret", "s", "", "Secret name")
	editCmd.Flags().StringP("version", "v", "latest", "Version to edit (default: latest)")
}

func runEdit(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	flagProject, _ := cmd.Flags().GetString("project")
	flagSecret, _ := cmd.Flags().GetString("secret")
	flagVersion, _ := cmd.Flags().GetString("version")

	projectID, err := selectProject(ctx, flagProject)
	if err != nil {
		return err
	}

	secretName, isNew, err := selectSecret(ctx, projectID, flagSecret, true)
	if err != nil {
		return err
	}

	// Fetch current content or start with empty JSON.
	var content []byte
	if isNew {
		content = []byte("{}\n")
	} else {
		content, err = gsm.GetSecretVersion(ctx, projectID, secretName, flagVersion)
		if err != nil {
			return err
		}
	}

	// Write to temp file.
	tmp, err := os.CreateTemp("", "mzcld-gsm-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck

	if _, err := tmp.Write(content); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmp.Close() //nolint:errcheck

	originalSum := sha256sum(content)

	// Edit loop with JSON validation.
	if err := editFileLoop(tmp.Name()); err != nil {
		return err
	}

	newContent, err := os.ReadFile(tmp.Name())
	if err != nil {
		return fmt.Errorf("failed to read edited file: %w", err)
	}

	newSum := sha256sum(newContent)
	if originalSum == newSum {
		ui.Info("No changes. Not pushing new version.")
		return nil
	}

	// Create the secret if it's new, then push the version.
	if isNew {
		ui.Info("Creating secret " + secretName + "...")
		if err := gsm.CreateSecret(ctx, projectID, secretName); err != nil {
			return err
		}
	}

	ui.Info("Pushing new version...")
	if err := gsm.AddSecretVersion(ctx, projectID, secretName, newContent); err != nil {
		return err
	}

	saveGSMCache(gsmCache{ProjectID: projectID, Secret: secretName})
	ui.Success(fmt.Sprintf("New version of %s pushed.", secretName))
	return nil
}

// editFileLoop opens the file in $EDITOR and validates JSON, looping on failure.
func editFileLoop(path string) error {
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
			return fmt.Errorf("aborted")
		}
	}
}

func sha256sum(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}
