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
	Use:     "info [<REPO>]",
	Aliases: []string{"get"},
	Example: "  gitclt repo info [-c CONNECTION] [-o OWNER] [--json] [<REPO>]",
	Short:   "Provides detailed information about a repository",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if _, e := repositories.RepositoryInfo(repo, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var repoDeleteCmd = &cobra.Command{
	Use:     "delete <REPO>",
	Aliases: []string{"rm", "remove"},
	Example: "  gitclt repo delete [-c CONNECTION] [-o OWNER] [-y] <REPO>",
	Short:   "Deletes a repository, after confirmation",
	Long: `This cannot be undone: the repository goes with everything in it. --yes skips the confirmation;
without a terminal to ask on, the repository is not deleted unless --yes is passed.`,
	Args: cobra.ExactArgs(1),
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
--default-branch (develop, where the forge can set it) -> GitHub cannot set it at creation, and uses your account's or organization's setting;
  Bitbucket can only set it with --initialize, as an empty repository has no branch yet. Passing the flag anyway is refused there
--public (true) -> repo visibility: true = public repo, private otherwise (--public=false)
--template (false) -> is this repo a template repo ?
--initialize (false) --> the repo will be created with basic .gitignore and README.md files
--actions (true) -> Actions enabled (Forgejo, GitHub); --actions=false creates the repository without it. Not sent to Bitbucket, unless passed

If the server keeps its own value (e.g. Forgejo with Actions turned off for the whole instance), the repository is still created,
with a warning
Other values are using hardcoded values such as gitignore template (Go, not available on Bitbucket), License (none supplied), etc`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := repositories.CreateRepo(args[0], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var repoEditCmd = &cobra.Command{
	Use:     "edit [<REPO>]",
	Aliases: []string{"modify", "update"},
	Example: "  gitclt repo edit [-c CONNECTION] [-o OWNER] [flags] [<REPO>]\n" +
		"  gitclt -c bitbucket repo edit --default-branch develop myrepo\n" +
		"  gitclt repo edit --archived --description \"\" oldrepo",
	Short: "Edits a repository; only the flags passed are changed",
	Long: `Only the fields whose flag is passed are sent to the server; everything else is left unchanged.
Boolean flags can be revoked with =false (e.g. --public=false makes the repository private),
and an empty string clears a text field (e.g. --website "").
--name renames the repository; the forge usually redirects the old URLs, but local clones should update their remotes.
Bitbucket cannot archive a repository, nor make it a template, and has none of the features and settings below.

Features (Forgejo and GitHub unless noted): --actions, --issues, --wiki, --pull-requests, --projects,
  --releases and --packages (Forgejo), --discussions (GitHub)
Merge settings: --merge-commit, --rebase, --squash, --delete-branch (both forges);
  --rebase-merge, --fast-forward-only, --rebase-update, --default-merge METHOD (Forgejo); --auto-merge, --update-branch (GitHub)
Issue tracker (Forgejo, needs issues on): --time-tracking, --time-tracking-contributors-only, --dependencies,
  --external-tracker URL [--tracker-format FORMAT] [--tracker-style STYLE] [--tracker-pattern REGEXP]; --external-tracker "" goes back
  to the built-in tracker, with its default settings (Forgejo does not keep them meanwhile)
Wiki (Forgejo): --external-wiki URL (needs the wiki on; "" goes back to the built-in wiki), --wiki-editable-by-all
Actions settings (GitHub): --actions-allowed all|local_only|selected, --actions-token read|write, --actions-approve-prs

A feature or setting the forge lacks is refused before any change. On Forgejo, merge settings need pull requests on.
If the server keeps its own value for something (e.g. a wiki on a free GitHub private repository), the rest is applied, with a warning.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if e := repositories.EditRepo(repo, cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	repoCmd.AddCommand(repoDeleteCmd, repoInfoCmd, repoListCmd, repoCreateCmd, repoEditCmd)

	repoCreateCmd.Flags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", "user, organization or workspace that will own the repository (default: the connection's owner, else your username)")
	repoCreateCmd.Flags().BoolVar(&repositories.TemplateRepo, "template", false, "template repo")
	repoCreateCmd.Flags().BoolVar(&repositories.InitializeRepo, "initialize", false, "initialize the repo after creation")
	repoCreateCmd.Flags().StringVar(&repositories.RepoDefaultBranch, "default-branch", "", "the default branch to use after creation (default: develop, where the forge can set it)")
	repoCreateCmd.Flags().StringVar(&repositories.RepoDescription, "description", "", "repository description")
	repoCreateCmd.Flags().BoolVar(&repositories.PublicRepo, "public", true, "public repository (use --public=false for a private one)")
	repoCreateCmd.Flags().BoolVar(&repositories.CreateActions, "actions", true, "enable Actions (use --actions=false to create the repository without it)")
	repoDeleteCmd.Flags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", "user, organization or workspace that owns the repository (default: the connection's owner, else your username)")
	repoDeleteCmd.Flags().BoolVarP(&repositories.AssumeYes, "yes", "y", false, "confirm deletion")
	repoInfoCmd.Flags().BoolVarP(&repositories.JSONoutput, "json", "j", false, "output as JSON")
	repoInfoCmd.Flags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", ownerFlagHelp)
	repoListCmd.Flags().BoolVarP(&repositories.JSONoutput, "json", "j", false, "output as JSON")
	repoListCmd.Flags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", "only show repositories owned by this user or organization (default: all accessible repositories)")
	repoListCmd.Flags().StringVarP(&repositories.SortBy, "sort", "s", "", "sort on this column, in descending order")
	repoListCmd.Flags().StringVarP(&repositories.SortByAsc, "sort-asc", "S", "", "sort on this column, in ascending order (default: ID)")

	repoEditCmd.Flags().StringVarP(&repositories.OwnerOrg, "owner", "o", "", ownerFlagHelp)
	repoEditCmd.Flags().StringVar(&repositories.EditRepoName, "name", "", "new name of the repository")
	repoEditCmd.Flags().StringVar(&repositories.EditRepoDescription, "description", "", "repository description")
	repoEditCmd.Flags().StringVar(&repositories.EditRepoWebsite, "website", "", "repository web site")
	repoEditCmd.Flags().StringVar(&repositories.EditRepoDefaultBranch, "default-branch", "", "default branch (it must exist)")
	repoEditCmd.Flags().BoolVar(&repositories.EditRepoPublic, "public", false, "public repository (--public=false makes it private)")
	repoEditCmd.Flags().BoolVar(&repositories.EditRepoArchived, "archived", false, "archive the repository (read-only); --archived=false unarchives it")
	repoEditCmd.Flags().BoolVar(&repositories.EditRepoTemplate, "template", false, "template repository (--template=false reverts it)")

	ef := repoEditCmd.Flags()
	ef.BoolVar(&repositories.EditActions, "actions", false, "Actions (--actions=false turns it off)")
	ef.BoolVar(&repositories.EditIssues, "issues", false, "issue tracker")
	ef.BoolVar(&repositories.EditWiki, "wiki", false, "wiki")
	ef.BoolVar(&repositories.EditPullRequests, "pull-requests", false, "pull requests")
	ef.BoolVar(&repositories.EditProjects, "projects", false, "projects")
	ef.BoolVar(&repositories.EditReleases, "releases", false, "releases (Forgejo)")
	ef.BoolVar(&repositories.EditPackages, "packages", false, "packages (Forgejo)")
	ef.BoolVar(&repositories.EditDiscussions, "discussions", false, "discussions (GitHub)")

	ef.BoolVar(&repositories.EditMergeCommit, "merge-commit", false, "allow merge commits")
	ef.BoolVar(&repositories.EditRebase, "rebase", false, "allow rebasing")
	ef.BoolVar(&repositories.EditRebaseMerge, "rebase-merge", false, "allow rebasing, then a merge commit (Forgejo)")
	ef.BoolVar(&repositories.EditSquash, "squash", false, "allow squashing")
	ef.BoolVar(&repositories.EditFastForwardOnly, "fast-forward-only", false, "allow fast-forward only merges (Forgejo)")
	ef.StringVar(&repositories.EditDefaultMerge, "default-merge", "", "merge method the web interface proposes: merge, rebase, rebase-merge, squash, fast-forward-only (Forgejo)")
	ef.BoolVar(&repositories.EditDeleteBranch, "delete-branch", false, "delete the head branch after a merge")
	ef.BoolVar(&repositories.EditAutoMerge, "auto-merge", false, "allow auto-merge (GitHub)")
	ef.BoolVar(&repositories.EditUpdateBranch, "update-branch", false, "offer to update a pull request's branch (GitHub)")
	ef.BoolVar(&repositories.EditRebaseUpdate, "rebase-update", false, "allow updating a pull request's branch by rebase (Forgejo)")

	ef.BoolVar(&repositories.EditTimeTracking, "time-tracking", false, "time tracking (Forgejo)")
	ef.BoolVar(&repositories.EditTimeTrackingContributorsOnly, "time-tracking-contributors-only", false, "only contributors track time (Forgejo)")
	ef.BoolVar(&repositories.EditDependencies, "dependencies", false, "dependencies between issues and pull requests (Forgejo)")
	ef.StringVar(&repositories.EditExternalTracker, "external-tracker", "", `external issue tracker URL; "" goes back to the built-in one (Forgejo)`)
	ef.StringVar(&repositories.EditTrackerFormat, "tracker-format", "", "external tracker's issue link, e.g. https://t.example.com/{user}/{repo}/{index}")
	ef.StringVar(&repositories.EditTrackerStyle, "tracker-style", "", "external tracker's issue numbers: numeric (default), alphanumeric, regexp")
	ef.StringVar(&repositories.EditTrackerPattern, "tracker-pattern", "", "external tracker's issue regexp, with --tracker-style regexp")
	ef.StringVar(&repositories.EditExternalWiki, "external-wiki", "", `external wiki URL; "" goes back to the built-in one (Forgejo)`)
	ef.BoolVar(&repositories.EditWikiEditableByAll, "wiki-editable-by-all", false, "every signed-in user can edit the wiki (Forgejo)")

	ef.StringVar(&repositories.EditActionsAllowed, "actions-allowed", "", "actions workflows may use: all, local_only, selected (GitHub)")
	ef.StringVar(&repositories.EditActionsToken, "actions-token", "", "default permission of the workflows' token: read, write (GitHub)")
	ef.BoolVar(&repositories.EditActionsApprovePRs, "actions-approve-prs", false, "workflows may approve pull requests (GitHub)")

	repoListCmd.MarkFlagsMutuallyExclusive("sort", "sort-asc")

	addHelp(repoInfoCmd, repoCommandInferenceHelp)
	addHelp(repoEditCmd, repoCommandInferenceHelp)
}
