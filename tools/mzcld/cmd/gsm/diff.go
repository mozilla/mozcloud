package gsm

import (
	"fmt"
	"strings"

	"charm.land/huh/v2/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/gsm"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Diff two versions of a secret",
	Long:  "Show a unified diff between two versions of a secret.",
	RunE:  runDiff,
}

func init() {
	diffCmd.Flags().StringP("project", "p", "", "GCP project ID")
	diffCmd.Flags().StringP("secret", "s", "", "Secret name")
	diffCmd.Flags().StringP("a", "a", "", "First version to compare (required)")
	diffCmd.Flags().StringP("b", "b", "", "Second version to compare (required)")
}

func runDiff(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	flagProject, _ := cmd.Flags().GetString("project")
	flagSecret, _ := cmd.Flags().GetString("secret")
	verA, _ := cmd.Flags().GetString("a")
	verB, _ := cmd.Flags().GetString("b")

	if verA == "" || verB == "" {
		return fmt.Errorf("both --a and --b version flags are required")
	}

	projectID, err := selectProject(ctx, flagProject)
	if err != nil {
		return err
	}

	secretName, _, err := selectSecret(ctx, projectID, flagSecret, false)
	if err != nil {
		return err
	}

	var dataA, dataB []byte
	var fetchErr error
	_ = spinner.New().
		Title(fmt.Sprintf("Fetching versions %s and %s...", verA, verB)).
		Context(ctx).
		Action(func() {
			dataA, fetchErr = gsm.GetSecretVersion(ctx, projectID, secretName, verA)
			if fetchErr != nil {
				return
			}
			dataB, fetchErr = gsm.GetSecretVersion(ctx, projectID, secretName, verB)
		}).
		Run()
	if fetchErr != nil {
		return fetchErr
	}

	linesA := strings.Split(string(dataA), "\n")
	linesB := strings.Split(string(dataB), "\n")

	diff := unifiedDiff(linesA, linesB, fmt.Sprintf("v%s", verA), fmt.Sprintf("v%s", verB))

	if len(diff) == 0 {
		ui.Info("No differences between versions.")
		return nil
	}

	add := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	del := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	hdr := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	for _, line := range diff {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"), strings.HasPrefix(line, "@@"):
			fmt.Println(hdr.Render(line))
		case strings.HasPrefix(line, "+"):
			fmt.Println(add.Render(line))
		case strings.HasPrefix(line, "-"):
			fmt.Println(del.Render(line))
		default:
			fmt.Println(line)
		}
	}

	cacheSelection(projectID, secretName)
	return nil
}

// unifiedDiff produces a simple unified diff between two sets of lines.
func unifiedDiff(a, b []string, nameA, nameB string) []string {
	// Use a simple LCS-based diff.
	lcs := lcsTable(a, b)

	var result []string
	result = append(result, fmt.Sprintf("--- %s", nameA))
	result = append(result, fmt.Sprintf("+++ %s", nameB))

	i, j := len(a), len(b)
	var patch []string
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			patch = append(patch, " "+a[i-1])
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			patch = append(patch, "+"+b[j-1])
			j--
		} else {
			patch = append(patch, "-"+a[i-1])
			i--
		}
	}

	// Reverse the patch (we built it bottom-up).
	for left, right := 0, len(patch)-1; left < right; left, right = left+1, right-1 {
		patch[left], patch[right] = patch[right], patch[left]
	}

	result = append(result, patch...)
	return result
}

func lcsTable(a, b []string) [][]int {
	m, n := len(a), len(b)
	table := make([][]int, m+1)
	for i := range table {
		table[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				table[i][j] = table[i-1][j-1] + 1
			} else if table[i-1][j] >= table[i][j-1] {
				table[i][j] = table[i-1][j]
			} else {
				table[i][j] = table[i][j-1]
			}
		}
	}
	return table
}
