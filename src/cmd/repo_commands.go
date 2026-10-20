// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/repo_commands.go
// Original timestamp : 2026.10.18 00:45:00

package cmd

import (
	"fmt"
	"gitclt/repositories"
	"gitclt/shared"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:     "repo",
	Aliases: []string{"repository"},
	Short:   "Repository commands",
}

var repoListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Example: "gitclt repo list [-c CONNECTION] [-o OWNER] [--json]",
	Short:   "Lists all repositories visible to the configured user",
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := repositories.ListRepositories(true); err != nil {
			fmt.Println(err.Error())
		}
	},
}

func init() {
	repoCmd.AddCommand(repoListCmd)

	repoCmd.PersistentFlags().StringVarP(&shared.ConnectionName, "connection", "c", "", "Connection to use (default: the default connection)")
	repoCmd.PersistentFlags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", "Only show repositories owned by this user or organization (default: all accessible repositories)")
	repoListCmd.Flags().BoolVarP(&repositories.RepoListJSONOutput, "json", "j", false, "Output the list as JSON")
	repoListCmd.Flags().StringVarP(&repositories.SortBy, "sort", "s", "", "Sort order")
}
