// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/token_commands.go
// Original timestamp : 2026.11.04 10:00:00

package cmd

import (
	"fmt"
	"gitclt/access_mgmt/tokens"

	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:     "token",
	Aliases: []string{"tokens"},
	Short:   "Access token management commands (Forgejo only)",
	Long: `Manages the access tokens of the connection's user; GitHub and Bitbucket have no API for this.
Forgejo refuses token authentication when creating or deleting a token: these commands use the connection's password,
and ask for it if the connection does not store one.
Forgejo cannot change an existing token: to change its permissions, create a new one, and delete the old one.`,
}

var tokenListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Example: "  gitclt token list [-c CONNECTION] [--json]",
	Short:   "Lists the access tokens of the connection's user (Forgejo only)",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := tokens.ListTokens(true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var tokenInfoCmd = &cobra.Command{
	Use:     "info <NAME>",
	Aliases: []string{"get"},
	Example: "  gitclt token info [-c CONNECTION] [--json] <NAME>\n" +
		"  gitclt token info --id 42",
	Short: "Gives information about an access token (Forgejo only)",
	Args:  tokenTargetArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := tokens.GetToken(tokenTarget(args), true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var tokenCreateCmd = &cobra.Command{
	Use:     "create <NAME>",
	Aliases: []string{"add"},
	Example: "  gitclt token create [-c CONNECTION] -s SCOPE [-s SCOPE] [-r OWNER/REPO] [--otp CODE] [--json] <NAME>\n" +
		"  gitclt token create -s read:user,write:repository -r mainline/test mytoken",
	Short: "Creates an access token, and shows its secret (Forgejo only)",
	Long: `The secret can only be seen here: it is shown once, and cannot be read back. With -q, only the secret is printed.
The scopes are passed to the server unchecked (ex: all, read:user, write:repository).
--repo limits the token to these repositories; without it, the scopes apply to all of them.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if e := tokens.CreateToken(args[0]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var tokenDeleteCmd = &cobra.Command{
	Use:     "delete <NAME>",
	Aliases: []string{"rm", "remove"},
	Example: "  gitclt token delete [-c CONNECTION] [--otp CODE] [--yes] <NAME>\n" +
		"  gitclt token delete --id 42",
	Short: "Deletes an access token (Forgejo only)",
	Args:  tokenTargetArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if e := tokens.DeleteToken(tokenTarget(args)); e != nil {
			fmt.Println(e.Error())
		}
	},
}

// A token is designated by its name, or by --id; not both
func tokenTargetArgs(cmd *cobra.Command, args []string) error {
	if !cmd.Flags().Changed("id") {
		return cobra.ExactArgs(1)(cmd, args)
	}
	if len(args) != 0 {
		return fmt.Errorf("pass either the token's name, or --id")
	}
	if tokens.TokenID <= 0 {
		return fmt.Errorf("--id must be a positive number")
	}
	return nil
}

func tokenTarget(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func init() {
	tokenCmd.AddCommand(tokenListCmd, tokenInfoCmd, tokenCreateCmd, tokenDeleteCmd)

	tokenListCmd.Flags().BoolVarP(&tokens.JSONoutput, "json", "j", false, "output as JSON")
	tokenInfoCmd.Flags().BoolVarP(&tokens.JSONoutput, "json", "j", false, "output as JSON")
	tokenInfoCmd.Flags().Int64Var(&tokens.TokenID, "id", 0, "designate the token by its ID instead of its name")

	tokenCreateCmd.Flags().StringSliceVarP(&tokens.CreateTokenScopes, "scope", "s", nil, "scope of the token (repeatable, or comma-separated)")
	tokenCreateCmd.Flags().StringSliceVarP(&tokens.CreateTokenRepos, "repo", "r", nil, "limit the token to this OWNER/REPO (repeatable, or comma-separated)")
	tokenCreateCmd.Flags().BoolVarP(&tokens.JSONoutput, "json", "j", false, "output as JSON")
	_ = tokenCreateCmd.MarkFlagRequired("scope")

	tokenDeleteCmd.Flags().Int64Var(&tokens.TokenID, "id", 0, "designate the token by its ID instead of its name")
	tokenDeleteCmd.Flags().BoolVarP(&tokens.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")
}
