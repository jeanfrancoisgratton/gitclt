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
		if _, e := repositories.ListRepositories(true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var repoInfoCmd = &cobra.Command{
	Use:     "info",
	Example: "gitclt repo info [-c CONNECTION] -o OWNER [--json] <REPO>",
	Short:   "Provides detailed information about a repository",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := repositories.RepositoryInfo(args[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var repoDeleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"rm"},
	Example: "gitclt repo delete [-c CONNECTION] -o OWNER [--json] <REPO>",
	Short:   "Delete a repository",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := repositories.DeleteRepo(args[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var repoCreateCmd = &cobra.Command{
	Long: `
Optional flags are below, with default values in parentheses:

--description () -> repo description
--defaultbranch (develop)
--publicrepo (true) -> repo visibility: true = public repo, private otherwise
--template (false) -> is this repo a template repo ?
--initialize (false) --> the repo will be created with basic .gitignore and README.md files

Other values are using hardcoded values such as gitignore template (Go, not available on Bitbucket), License (none supplied), etc`,
	Use:     "create",
	Example: "gitclt repo create [-c CONNECTION] -o OWNER [OTHER FLAGS] <REPO>",
	Short:   "Create a repository",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := repositories.CreateRepo(args[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	repoCmd.AddCommand(repoDeleteCmd, repoInfoCmd, repoListCmd, repoCreateCmd)

	repoCmd.PersistentFlags().StringVarP(&shared.ConnectionName, "connection", "c", "", "Connection to use (default: the default connection)")
	repoCreateCmd.Flags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", "User or organization that will own the repository")
	repoCreateCmd.Flags().BoolVar(&repositories.TemplateRepo, "template", false, "Template repo")
	repoCreateCmd.Flags().BoolVar(&repositories.InitializeRepo, "initialize", false, "Initialize the repo after creation")
	repoCreateCmd.Flags().StringVar(&repositories.RepoDefaultBranch, "defaultbranch", "develop", "The default branch to use after creation")
	repoCreateCmd.Flags().StringVar(&repositories.RepoDescription, "description", "", "Repository description")
	repoCreateCmd.Flags().BoolVar(&repositories.PublicRepo, "public", true, "Public repository (use --publicrepo=false for a private one)")
	repoDeleteCmd.Flags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", "User or organization that owns the repository")
	repoInfoCmd.Flags().BoolVarP(&repositories.JSONoutput, "json", "j", false, "Output the list as JSON")
	repoInfoCmd.Flags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", "User or organization that owns the repository")
	repoListCmd.Flags().BoolVarP(&repositories.JSONoutput, "json", "j", false, "Output the list as JSON")
	repoListCmd.Flags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", "Only show repositories owned by this user or organization (default: all accessible repositories)")
	repoListCmd.Flags().StringVarP(&repositories.SortBy, "sort", "s", "", "Sort on this column, in descending order")
	repoListCmd.Flags().StringVarP(&repositories.SortByAsc, "sort-asc", "S", "", "Sort on this column, in ascending order (default: ID)")

	repoListCmd.MarkFlagsMutuallyExclusive("sort", "sort-asc")
	repoCreateCmd.MarkFlagRequired("owner")
	repoDeleteCmd.MarkFlagRequired("owner")
	repoInfoCmd.MarkFlagRequired("owner")

}
