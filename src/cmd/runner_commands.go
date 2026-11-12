// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/runner_commands.go
// Original timestamp : 2026.11.11 18:00:00

package cmd

import (
	"fmt"
	"gitclt/ci_mgmt/runners"

	"github.com/spf13/cobra"
)

var runnerCmd = &cobra.Command{
	Use:     "runner",
	Aliases: []string{"runners"},
	Short:   "CI runner commands: the forge's records of the runners (Forgejo, GitHub)",
	Long: `The runners the forge knows of, for Forgejo Actions and GitHub Actions self-hosted runners; Bitbucket is not supported.
This is only the forge's side: installing and starting a runner's program (forgejo-runner, GitHub's runner) is left to
other tools, which take what runner register or runner token print.
The runners are a repository's by default; --org ORG takes an organization's, --user the connection's user's (Forgejo),
and --instance the whole instance's (Forgejo site admins only). A runner is only reached through its own scope.`,
}

var runnerListCmd = &cobra.Command{
	Use:     "list [<REPO>]",
	Aliases: []string{"ls"},
	Example: "  gitclt runner list [-c CONNECTION] [-o OWNER] [--org ORG | --user | --instance] [--inherited] [-j] [<REPO>]\n" +
		"  gitclt runner ls --org devops",
	Short: "Lists the runners; -q outputs only their IDs",
	Long: `State is the same word on every forge: offline, idle or busy. runner get also shows the forge's own status.
--inherited adds the runners the scope can use from above it, such as a repository's organization's (Forgejo only).`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if _, e := runners.ListRunners(repo, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runnerGetCmd = &cobra.Command{
	Use:     "get [<REPO>] <RUNNER-ID>",
	Aliases: []string{"info", "show"},
	Example: "  gitclt runner get [-c CONNECTION] [-o OWNER] [--org ORG | --user | --instance] [-j] [<REPO>] <RUNNER-ID>",
	Short:   "Shows a runner",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := splitRepoNumber(args, 1)
		if _, e := runners.GetRunner(repo, rest[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runnerRegisterCmd = &cobra.Command{
	Use:     "register [<REPO>]",
	Aliases: []string{"add", "create"},
	Example: "  gitclt runner register [-c CONNECTION] [-o OWNER] [--org ORG | --user | --instance] --name NAME [--description TEXT] [--ephemeral] [--label LABEL,...] [<REPO>]\n" +
		"  gitclt runner register --instance --name build-1\n" +
		"  gitclt -c github runner register --name ci-once --ephemeral --label self-hosted,linux myrepo",
	Short: "Registers a runner, and prints what its program needs to connect",
	Long: `This creates the runner's record on the forge; the runner's program still has to be installed and started.
On Forgejo, it prints the ID, UUID and token of the runner's .runner file. The runner declares its own labels, so
--label is refused, and names may repeat: registering twice makes two runners.
On GitHub, only single-use (just-in-time) runners can be registered: --ephemeral is required, --description is refused,
and the labels default to self-hosted. It prints the just-in-time configuration the runner's run.sh starts with.
For a lasting runner, use
gitclt runner token and the runner's own configuration script.
The secret is shown only this once; -q prints it alone, for scripts.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if e := runners.RegisterRunner(repo); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runnerTokenCmd = &cobra.Command{
	Use:     "token [<REPO>]",
	Aliases: []string{"registration-token"},
	Example: "  gitclt runner token [-c CONNECTION] [-o OWNER] [--org ORG | --user | --instance] [<REPO>]\n" +
		"  gitclt -q runner token --instance   # the token alone, for forgejo-runner register",
	Short: "Prints a registration token, for a runner's program to register with",
	Long: `Forgejo's registration token does not expire; GitHub's expires within the hour, and its expiry is shown.
-q prints the token alone, for scripts.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := splitRepo(args, 0)
		if e := runners.RunnerToken(repo); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var runnerDeleteCmd = &cobra.Command{
	Use:     "delete [<REPO>] <RUNNER-ID>...",
	Aliases: []string{"rm", "del"},
	Example: "  gitclt runner delete [-c CONNECTION] [-o OWNER] [--org ORG | --user | --instance] [-y] [<REPO>] <RUNNER-ID>...",
	Short:   "Deletes runners' records, after confirmation",
	Long: `Every runner is checked before asking. The run stops at the first failure; the runners deleted before it stay deleted.
The runner's program is left alone: it can no longer connect, and is stopped or removed elsewhere.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, ids := splitRepoNumber(args, 1)
		if e := runners.DeleteRunners(repo, ids); e != nil {
			fmt.Println(e.Error())
		}
	},
}

const runnerIDHelp = `An ID is never taken for <REPO>, so several need no . (gitclt runner rm 3 4);
a repository with a numeric name cannot be designated by this command.`

func init() {
	runnerCmd.AddCommand(runnerListCmd, runnerGetCmd, runnerRegisterCmd, runnerTokenCmd, runnerDeleteCmd)

	pf := runnerCmd.PersistentFlags()
	pf.StringVarP(&runners.OwnerOrg, "owner", "o", "", ownerFlagHelp)
	pf.StringVar(&runners.OrgScope, "org", "", "an organization's runners, instead of a repository's")
	pf.BoolVar(&runners.UserScope, "user", false, "the connection's user's runners, instead of a repository's (Forgejo)")
	pf.BoolVar(&runners.InstanceScope, "instance", false, "the whole instance's runners, instead of a repository's (Forgejo site admins)")
	runnerCmd.MarkFlagsMutuallyExclusive("org", "user", "instance")

	for _, c := range []*cobra.Command{runnerListCmd, runnerGetCmd} {
		c.Flags().BoolVarP(&runners.JSONoutput, "json", "j", false, "output as JSON")
	}
	runnerListCmd.Flags().BoolVar(&runners.Inherited, "inherited", false, "add the runners the scope can use from above it (Forgejo)")

	runnerRegisterCmd.Flags().StringVar(&runners.RegisterName, "name", "", "the runner's name (required)")
	runnerRegisterCmd.Flags().StringVar(&runners.RegisterDescription, "description", "", "the runner's description (Forgejo)")
	runnerRegisterCmd.Flags().BoolVar(&runners.RegisterEphemeral, "ephemeral", false, "a single-use runner: it takes one job, then goes away (required on GitHub)")
	runnerRegisterCmd.Flags().StringSliceVar(&runners.RegisterLabels, "label", nil, "labels the runner's jobs ask for, comma-separated (GitHub; default: self-hosted)")

	runnerDeleteCmd.Flags().BoolVarP(&runners.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")

	addHelp(runnerCmd, repoInferenceHelp)
	for _, c := range []*cobra.Command{runnerGetCmd, runnerDeleteCmd} {
		addHelp(c, runnerIDHelp)
	}
}
