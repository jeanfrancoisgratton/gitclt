// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/ci_commands.go
// Original timestamp : 2026.11.06 18:00:00

package cmd

import (
	"fmt"
	"gitclt/ci_mgmt/ci"

	"github.com/spf13/cobra"
)

const ciScopeHelp = `They belong to a repository, unless a flag says otherwise; with --org or --user, <REPO> is left out:
  --org ORG   an organization, or a workspace on Bitbucket, for all its repositories
  --user      your own account (Forgejo only)
  --env ENV   an environment of the repository (GitHub, Bitbucket), which must already exist`

// ciArgs validates [<REPO>] followed by n arguments, or at least n; --org and --user leave no room for a repository
func ciArgs(n int, atLeast bool) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		switch {
		case atLeast:
			return cobra.MinimumNArgs(n)(cmd, args)
		case ci.HasOwnScope():
			return cobra.ExactArgs(n)(cmd, args)
		}
		return cobra.RangeArgs(n, n+1)(cmd, args)
	}
}

func ciSplit(args []string, n int) (string, []string) {
	if ci.HasOwnScope() {
		return "", args
	}
	return splitRepo(args, n)
}

var secretCmd = &cobra.Command{
	Use:     "secret",
	Aliases: []string{"secrets"},
	Short:   "CI secret commands: Forgejo Actions, GitHub Actions, Bitbucket Pipelines",
	Long: `Secrets are write-only: once stored, they can be replaced or deleted, never read back.
Forgejo and GitHub upper-case the names; Bitbucket keeps them as given. Names are letters, digits and underscores, not starting
with a digit, nor, on Forgejo, with GITEA_, GITHUB_ or FORGEJO_.
` + ciScopeHelp,
}

var secretListCmd = &cobra.Command{
	Use:     "list [<REPO>]",
	Aliases: []string{"ls"},
	Example: "  gitclt secret list [-c CONNECTION] [-o OWNER] [--org ORG | --user | --env ENV] [-j] [<REPO>]",
	Short:   "Lists the secrets' names; -q outputs only those",
	Long:    "Their values cannot be read back. Forgejo before 17 cannot list your own secrets (--user).",
	Args:    ciArgs(0, false),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := ciSplit(args, 0)
		if _, e := ci.ListSecrets(repo, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var secretSetCmd = &cobra.Command{
	Use:     "set [<REPO>] <NAME>",
	Aliases: []string{"add", "create", "update"},
	Example: "  gitclt secret set [-c CONNECTION] [-o OWNER] [--org ORG | --user | --env ENV] [-f FILE | --stdin] [<REPO>] <NAME>\n" +
		"  gitclt secret set DEPLOY_TOKEN\n" +
		"  gitclt secret set --stdin DEPLOY_TOKEN < token.txt",
	Short: "Creates a secret, or replaces its value without asking",
	Long: `The value is never an argument, where any local user could read it: it is prompted for, without being shown;
-f, --from-file reads it from a file only you can access; --stdin reads it from the standard input. A trailing line ending is dropped.
A secret's old value cannot be read back, so a replaced one is gone. Forgejo turns line endings into LF: base64-encode a value whose exact bytes matter.
A new secret of a GitHub organization is available to all its repositories; an existing one keeps its access.`,
	Args: ciArgs(1, false),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := ciSplit(args, 1)
		if e := ci.SetSecret(repo, rest[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var secretDeleteCmd = &cobra.Command{
	Use:     "delete [<REPO>] <NAME>...",
	Aliases: []string{"rm", "remove", "del"},
	Example: "  gitclt secret delete [-c CONNECTION] [-o OWNER] [--org ORG | --user | --env ENV] [-y] [<REPO>] <NAME>...\n" +
		"  gitclt secret rm . OLD_TOKEN OLDER_TOKEN",
	Short: "Deletes secrets, after confirmation",
	Long: `Every name is checked before asking, and the question names where they live.
The run stops at the first failure; the secrets deleted before it stay deleted.`,
	Args: ciArgs(1, true),
	Run: func(cmd *cobra.Command, args []string) {
		repo, names := ciSplit(args, 1)
		if e := ci.DeleteSecrets(repo, names); e != nil {
			printTargetsError(e)
		}
	},
}

var variableCmd = &cobra.Command{
	Use:     "variable",
	Aliases: []string{"variables", "var", "vars"},
	Short:   "CI variable commands: Forgejo Actions, GitHub Actions, Bitbucket Pipelines",
	Long: `Variables are like secrets, but readable by anyone who can read the settings: they are not for credentials.
Forgejo and GitHub upper-case the names; Bitbucket keeps them as given. Names are letters, digits and underscores, not starting
with a digit, nor, on Forgejo, with GITEA_, GITHUB_ or FORGEJO_.
` + ciScopeHelp,
}

var variableListCmd = &cobra.Command{
	Use:     "list [<REPO>]",
	Aliases: []string{"ls"},
	Example: "  gitclt variable list [-c CONNECTION] [-o OWNER] [--org ORG | --user | --env ENV] [-j] [<REPO>]",
	Short:   "Lists the variables, with the beginning of their values; -q outputs only their names",
	Args:    ciArgs(0, false),
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := ciSplit(args, 0)
		if _, e := ci.ListVariables(repo, true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var variableGetCmd = &cobra.Command{
	Use:     "get [<REPO>] <NAME>",
	Aliases: []string{"info"},
	Example: "  gitclt variable get [-c CONNECTION] [-o OWNER] [--org ORG | --user | --env ENV] [-j] [<REPO>] <NAME>",
	Short:   "Shows a variable, with its whole value; -q outputs only the value",
	Args:    ciArgs(1, false),
	Run: func(cmd *cobra.Command, args []string) {
		repo, rest := ciSplit(args, 1)
		if _, e := ci.GetVariable(repo, rest[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var variableSetCmd = &cobra.Command{
	Use:     "set [<REPO>] <NAME> [<VALUE>]",
	Aliases: []string{"add", "create", "update"},
	Example: "  gitclt variable set [-c CONNECTION] [-o OWNER] [--org ORG | --user | --env ENV] [<REPO>] <NAME> <VALUE>\n" +
		"  gitclt variable set REGION eu\n" +
		"  gitclt variable set -f config.yaml CONFIG",
	Short: "Creates a variable, or changes its value",
	Long: `The value is an argument, or comes from -f, --from-file or --stdin instead; a trailing line ending is then dropped.
Empty values are refused. Bitbucket keeps secrets and variables in one list: a name used by one kind cannot be set as the other.`,
	Args: func(cmd *cobra.Command, args []string) error {
		n := 2
		if cmd.Flags().Changed("from-file") || cmd.Flags().Changed("stdin") {
			n = 1
		}
		return ciArgs(n, false)(cmd, args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		n, value := 2, ""
		if cmd.Flags().Changed("from-file") || cmd.Flags().Changed("stdin") {
			n = 1
		}
		repo, rest := ciSplit(args, n)
		if n == 2 {
			value = rest[1]
		}
		if e := ci.SetVariable(repo, rest[0], value); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var variableDeleteCmd = &cobra.Command{
	Use:     "delete [<REPO>] <NAME>...",
	Aliases: []string{"rm", "remove", "del"},
	Example: "  gitclt variable delete [-c CONNECTION] [-o OWNER] [--org ORG | --user | --env ENV] [-y] [<REPO>] <NAME>...",
	Short:   "Deletes variables, after confirmation",
	Long: `Every name is checked before asking, and the question names where they live.
The run stops at the first failure; the variables deleted before it stay deleted.`,
	Args: ciArgs(1, true),
	Run: func(cmd *cobra.Command, args []string) {
		repo, names := ciSplit(args, 1)
		if e := ci.DeleteVariables(repo, names); e != nil {
			printTargetsError(e)
		}
	},
}

func init() {
	secretCmd.AddCommand(secretListCmd, secretSetCmd, secretDeleteCmd)
	variableCmd.AddCommand(variableListCmd, variableGetCmd, variableSetCmd, variableDeleteCmd)

	for _, parent := range []*cobra.Command{secretCmd, variableCmd} {
		parent.PersistentFlags().StringVarP(&ci.OwnerOrg, "owner", "o", "", ownerFlagHelp)
		parent.PersistentFlags().StringVar(&ci.ScopeOrg, "org", "", "those of this organization, or workspace on Bitbucket")
		parent.PersistentFlags().BoolVar(&ci.ScopeUser, "user", false, "your own (Forgejo only)")
		parent.PersistentFlags().StringVar(&ci.ScopeEnv, "env", "", "those of this environment of the repository (GitHub, Bitbucket)")
		parent.MarkFlagsMutuallyExclusive("org", "user", "env")
		parent.MarkFlagsMutuallyExclusive("org", "owner")
		parent.MarkFlagsMutuallyExclusive("user", "owner")
		addHelp(parent, repoInferenceHelp)
	}
	secretListCmd.Flags().BoolVarP(&ci.JSONoutput, "json", "j", false, "output as JSON")
	secretSetCmd.Flags().StringVarP(&ci.ValueFile, "from-file", "f", "", "read the value from this file, which only you can access")
	secretSetCmd.Flags().BoolVar(&ci.ValueStdin, "stdin", false, "read the value from the standard input")
	secretSetCmd.MarkFlagsMutuallyExclusive("from-file", "stdin")
	secretDeleteCmd.Flags().BoolVarP(&ci.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")

	variableListCmd.Flags().BoolVarP(&ci.JSONoutput, "json", "j", false, "output as JSON")
	variableGetCmd.Flags().BoolVarP(&ci.JSONoutput, "json", "j", false, "output as JSON")
	variableSetCmd.Flags().StringVarP(&ci.ValueFile, "from-file", "f", "", "read the value from this file")
	variableSetCmd.Flags().BoolVar(&ci.ValueStdin, "stdin", false, "read the value from the standard input")
	variableSetCmd.MarkFlagsMutuallyExclusive("from-file", "stdin")
	variableDeleteCmd.Flags().BoolVarP(&ci.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")
}
