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
	Long: `The -o flag will filter repos belonging to that organization.
The -s flag will sort the list in descending order according to the specified sort criteria; -S does the same in ascending order.
Without -s or -S, the list is sorted by ID in ascending order.
To select a column, just type in its first letter.`,
}

var repoListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Example: "gitclt repo list [-c CONNECTION] [-o OWNER] [-s|-S SORT_CRITERIA] [--json]",
	Short:   "Lists all repositories visible to the configured user",
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := repositories.ListRepositories(true); err != nil {
			fmt.Println(err.Error())
		}
	},
}

var repoInfoCmd = &cobra.Command{
	Use:     "info",
	Example: "gitclt repo info [-c CONNECTION] [-o OWNER] [--json] <REPO>",
	Short:   "Provides detailed information about a repository",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := repositories.RepositoryInfo(args[0], true); err != nil {
			fmt.Println(err.Error())
		}
	},
}

func init() {
	repoCmd.AddCommand(repoListCmd)

	repoCmd.PersistentFlags().StringVarP(&shared.ConnectionName, "connection", "c", "", "Connection to use (default: the default connection)")
	repoCmd.PersistentFlags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", "Only show repositories owned by this user or organization (default: all accessible repositories)")
	repoListCmd.Flags().BoolVarP(&repositories.RepoListJSONOutput, "json", "j", false, "Output the list as JSON")
	repoListCmd.Flags().StringVarP(&repositories.SortBy, "sort", "s", "", "Sort on this column, in descending order")
	repoListCmd.Flags().StringVarP(&repositories.SortByAsc, "sort-asc", "S", "", "Sort on this column, in ascending order (default: ID)")
	repoListCmd.MarkFlagsMutuallyExclusive("sort", "sort-asc")
}
