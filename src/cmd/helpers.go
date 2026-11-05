// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/helpers.go
// Original timestamp : 2026.10.25 20:31:32

package cmd

import (
	"fmt"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	"github.com/spf13/cobra"
)

// completeVisibility offers the values shared.VisibilityFlag accepts, for any --visibility flag
func completeVisibility(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{"public", "limited", "private"}, cobra.ShellCompDirectiveNoFileComp
}

// A command's positional arguments are [<REPO>] followed by n others; without a repository, it is the one the current
// directory is a clone of
// A variable number of arguments cannot tell a repository from a target: past n, the first one is always the repository,
// as it always was, and "." stands for the current clone's, as in: gitclt tag rm . v1 v2
func splitRepo(args []string, n int) (string, []string) {
	if len(args) == n {
		return "", args
	}
	if args[0] == "." {
		return "", args[1:]
	}
	return args[0], args[1:]
}

// splitRepoNumber is splitRepo for commands whose first target is a pull request number: a number is never taken for
// the repository, so several pull requests need no "."
func splitRepoNumber(args []string, n int) (string, []string) {
	if isNumber(args[0]) {
		return "", args
	}
	return splitRepo(args, n)
}

// A pull request number has an optional leading #
func isNumber(s string) bool {
	s = strings.TrimPrefix(s, "#")
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

// printTargetsError prints an error of a command with several targets; a repository that does not exist there is most
// likely a first target, from someone inside a clone
func printTargetsError(e *cerr.CustomError) {
	if e.Title == "No such repository" || e.Title == "Not the same repository" {
		e.Message += "; the first of several arguments is the repository: from inside a clone, pass . for the current one"
	}
	fmt.Println(e.Error())
}

const ownerFlagHelp = "user, organization or workspace that owns the repository (default: the current clone's, for its repository; else the connection's owner, else your username)"

const repoInferenceHelp = `From inside a clone (anywhere in it), <REPO> and -o can be left out: they are read from the remote that is on the
connection's server (origin if several are), so -c picks which forge is asked about. Outside a clone, or with no remote on
that server, the command stops and asks for <REPO>. -o still overrides the owner.
A repository named without -o takes the clone's owner if it is the clone's own; any other takes the connection's owner, else your username.
A command taking several targets reads the first of two or more as <REPO>; use . for the current clone's: gitclt tag rm . v1 v2`

// The same, for the commands on the repository itself, which have no other argument
const repoCommandInferenceHelp = `From inside a clone (anywhere in it), <REPO> and -o can be left out: they are read from the remote that is on the
connection's server (origin if several are), so -c picks which forge is asked about. Outside a clone, or with no remote on
that server, the command stops and asks for <REPO>. -o still overrides the owner.
A repository named without -o takes the clone's owner if it is the clone's own; any other takes the connection's owner, else your username.`

const prNumberHelp = `A number is never taken for <REPO>, so several pull requests need no . (gitclt pr close 12 13);
a repository with a numeric name cannot be designated by this command.`

const confirmNamesRepoHelp = `The confirmation question names the repository, as it may not have been typed.`

// addHelp appends a paragraph to a command's help, and to its subcommands'; Long replaces Short in --help, so a command
// without one starts from its Short
func addHelp(cmd *cobra.Command, help string) {
	long := cmd.Long
	if long == "" {
		long = cmd.Short
	}
	if !strings.Contains(long, help) {
		cmd.Long = strings.TrimRight(long, "\n") + "\n\n" + help
	}
	for _, sub := range cmd.Commands() {
		addHelp(sub, help)
	}
}
