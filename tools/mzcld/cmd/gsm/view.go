package gsm

import (
	"encoding/json"
	"fmt"

	"github.com/mozilla/mozcloud/tools/mzcld/internal/gsm"
	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "View a secret's content",
	Long:  "Fetch and display the content of a secret version.",
	RunE:  runView,
}

func init() {
	viewCmd.Flags().StringP("project", "p", "", "GCP project ID")
	viewCmd.Flags().StringP("secret", "s", "", "Secret name")
	viewCmd.Flags().StringP("version", "v", "latest", "Version to view (default: latest)")
}

func runView(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	flagProject, _ := cmd.Flags().GetString("project")
	flagSecret, _ := cmd.Flags().GetString("secret")
	flagVersion, _ := cmd.Flags().GetString("version")

	projectID, err := selectProject(ctx, flagProject)
	if err != nil {
		return err
	}

	client, err := gsm.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close() //nolint:errcheck

	secretName, _, err := selectSecret(ctx, client, projectID, flagSecret, false)
	if err != nil {
		return err
	}

	var data []byte
	runWithSpinner(ctx, "Fetching secret...", func() {
		data, err = client.GetSecretVersion(ctx, projectID, secretName, flagVersion)
	})
	if err != nil {
		return err
	}

	// Pretty-print if the content is valid JSON.
	var pretty json.RawMessage
	if json.Unmarshal(data, &pretty) == nil {
		formatted, err := json.MarshalIndent(pretty, "", "  ")
		if err == nil {
			fmt.Println(string(formatted))
			cacheSelection(projectID, secretName)
			return nil
		}
	}

	fmt.Print(string(data))
	cacheSelection(projectID, secretName)
	return nil
}
