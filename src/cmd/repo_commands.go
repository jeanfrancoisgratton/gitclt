// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/repo_commands.go
// Original timestamp : 2026.10.18 00:45:00

package cmd

import (
	"fmt"
	"gitclt/repo_mgmt/repositories"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:     "repo",
	Aliases: []string{"repository"},
	Short:   "Repository commands",
	Long: `The -o flag gives the user, organization or workspace that owns the repository; it defaults to the connection's owner,
else your username. With repo list, -o instead filters the list (default: all the repositories you can see).
The -s flag will sort the list in descending order according to the specified sort criteria; -S does the same in ascending order.
Without -s or -S, the list is sorted by ID in ascending order.
To select a column, just type in its first letter.`,
}

var repoListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Example: "  gitclt repo list [-c CONNECTION] [-o OWNER] [-s|-S SORT_CRITERIA] [--json]",
	Short:   "Lists all repositories visible to the configured user",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := repositories.ListRepositories(true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var repoInfoCmd = &cobra.Command{
	Use:     "info <REPO>",
	Aliases: []string{"get"},
	Example: "  gitclt repo info [-c CONNECTION] [-o OWNER] [--json] <REPO>",
	Short:   "Provides detailed information about a repository",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := repositories.RepositoryInfo(args[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var repoDeleteCmd = &cobra.Command{
	Use:     "delete <REPO>",
	Aliases: []string{"rm", "remove"},
	Example: "  gitclt repo delete [-c CONNECTION] [-o OWNER] <REPO>",
	Short:   "Deletes a repository",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := repositories.DeleteRepo(args[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var repoCreateCmd = &cobra.Command{
	Use:     "create <REPO>",
	Aliases: []string{"add"},
	Example: "  gitclt repo create [-c CONNECTION] [-o OWNER] [flags] <REPO>",
	Short:   "Creates a repository",
	Long: `
Optional flags are below, with default values in parentheses:

--description () -> repo description
--default-branch (develop)
--public (true) -> repo visibility: true = public repo, private otherwise (--public=false)
--template (false) -> is this repo a template repo ?
--initialize (false) --> the repo will be created with basic .gitignore and README.md files

Other values are using hardcoded values such as gitignore template (Go, not available on Bitbucket), License (none supplied), etc`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := repositories.CreateRepo(args[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var repoEditCmd = &cobra.Command{
	Use:     "edit",
	Aliases: []string{"modify", "update"},
	Example: "  gitclt repo edit [-c CONNECTION] [-o OWNER] [flags] <REPO>\n" +
		"  gitclt -c bitbucket repo edit --default-branch develop myrepo\n" +
		"  gitclt repo edit --archived --description \"\" oldrepo",
	Short: "Edits a repository; only the flags passed are changed",
	Long: `Only the fields whose flag is passed are sent to the server; everything else is left unchanged.
Boolean flags can be revoked with =false (e.g. --public=false makes the repository private),
and an empty string clears a text field (e.g. --website "").
--name renames the repository; the forge usually redirects the old URLs, but local clones should update their remotes.
Bitbucket cannot archive a repository, nor make it a template.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := repositories.EditRepo(args[0], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	repoCmd.AddCommand(repoDeleteCmd, repoInfoCmd, repoListCmd, repoCreateCmd, repoEditCmd)

	repoCreateCmd.Flags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", "user, organization or workspace that will own the repository (default: the connection's owner, else your username)")
	repoCreateCmd.Flags().BoolVar(&repositories.TemplateRepo, "template", false, "template repo")
	repoCreateCmd.Flags().BoolVar(&repositories.InitializeRepo, "initialize", false, "initialize the repo after creation")
	repoCreateCmd.Flags().StringVar(&repositories.RepoDefaultBranch, "default-branch", "develop", "the default branch to use after creation")
	repoCreateCmd.Flags().StringVar(&repositories.RepoDescription, "description", "", "repository description")
	repoCreateCmd.Flags().BoolVar(&repositories.PublicRepo, "public", true, "public repository (use --public=false for a private one)")
	repoDeleteCmd.Flags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", "user, organization or workspace that owns the repository (default: the connection's owner, else your username)")
	repoInfoCmd.Flags().BoolVarP(&repositories.JSONoutput, "json", "j", false, "output as JSON")
	repoInfoCmd.Flags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", "user, organization or workspace that owns the repository (default: the connection's owner, else your username)")
	repoListCmd.Flags().BoolVarP(&repositories.JSONoutput, "json", "j", false, "output as JSON")
	repoListCmd.Flags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", "only show repositories owned by this user or organization (default: all accessible repositories)")
	repoListCmd.Flags().StringVarP(&repositories.SortBy, "sort", "s", "", "sort on this column, in descending order")
	repoListCmd.Flags().StringVarP(&repositories.SortByAsc, "sort-asc", "S", "", "sort on this column, in ascending order (default: ID)")

	repoEditCmd.Flags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", "user, organization or workspace that owns the repository (default: the connection's owner, else your username)")
	repoEditCmd.Flags().StringVar(&repositories.EditRepoName, "name", "", "new name of the repository")
	repoEditCmd.Flags().StringVar(&repositories.EditRepoDescription, "description", "", "repository description")
	repoEditCmd.Flags().StringVar(&repositories.EditRepoWebsite, "website", "", "repository web site")
	repoEditCmd.Flags().StringVar(&repositories.EditRepoDefaultBranch, "default-branch", "", "default branch (it must exist)")
	repoEditCmd.Flags().BoolVar(&repositories.EditRepoPublic, "public", false, "public repository (--public=false makes it private)")
	repoEditCmd.Flags().BoolVar(&repositories.EditRepoArchived, "archived", false, "archive the repository (read-only); --archived=false unarchives it")
	repoEditCmd.Flags().BoolVar(&repositories.EditRepoTemplate, "template", false, "template repository (--template=false reverts it)")

	repoListCmd.MarkFlagsMutuallyExclusive("sort", "sort-asc")
}
