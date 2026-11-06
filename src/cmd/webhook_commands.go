// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/webhook_commands.go
// Original timestamp : 2026.11.06 09:00:00

package cmd

import (
	"fmt"
	"gitclt/repo_mgmt/webhooks"

	"github.com/spf13/cobra"
)

var webhookCmd = &cobra.Command{
	Use:     "webhook",
	Aliases: []string{"webhooks", "hook", "hooks"},
	Short:   "Webhook commands: your own, the server's, a repository's or an organization's",
	Long: `A webhook sends an HTTP request to a URL when events happen.
  user     your own webhooks, for the events of your repositories (Forgejo only)
  system   the webhooks of the whole server, for the events of every repository (Forgejo only, needs admin rights)
  default  templates copied into each repository created afterwards (Forgejo only, needs admin rights)
  repo     the webhooks of a repository
  org      the webhooks of an organization (a workspace on Bitbucket)
A webhook's URL is shown as it is: for Slack or Mattermost, that URL is a credential.`,
}

// hookScope is one of the webhook kinds: they all have the same commands, except for what the API cannot do
type hookScope struct {
	name    string
	aliases []string
	short   string
	long    string
	prefix  string // what comes before the other arguments
	noList  bool
	ping    bool
	// The validator of a command taking n arguments of its own, or at least n
	args func(n int, atLeast bool) cobra.PositionalArgs
	// Splits the arguments into the target and the command's own; numbers: the first of them is a webhook ID
	split func(args []string, n int, numbers bool) (webhooks.Target, []string)
}

func plainArgs(n int, atLeast bool) cobra.PositionalArgs {
	if atLeast {
		return cobra.MinimumNArgs(n)
	}
	return cobra.ExactArgs(n)
}

func repoArgs(n int, atLeast bool) cobra.PositionalArgs {
	if atLeast {
		return cobra.MinimumNArgs(n)
	}
	return cobra.RangeArgs(n, n+1)
}

func orgArgs(n int, atLeast bool) cobra.PositionalArgs { return plainArgs(n+1, atLeast) }

func plainSplit(t webhooks.Target) func([]string, int, bool) (webhooks.Target, []string) {
	return func(args []string, _ int, _ bool) (webhooks.Target, []string) { return t, args }
}

func repoSplit(args []string, n int, numbers bool) (webhooks.Target, []string) {
	if numbers {
		repo, rest := splitRepoNumber(args, n)
		return webhooks.Repo(repo), rest
	}
	repo, rest := splitRepo(args, n)
	return webhooks.Repo(repo), rest
}

func orgSplit(args []string, _ int, _ bool) (webhooks.Target, []string) {
	return webhooks.Org(args[0]), args[1:]
}

var hookScopes = []hookScope{
	{name: "user", short: "Your own webhooks, for the events of your repositories (Forgejo only)",
		args: plainArgs, split: plainSplit(webhooks.User())},
	{name: "system", short: "The server's webhooks, for the events of every repository (Forgejo only, needs admin rights)",
		long: `Forgejo's API only creates these through an undocumented setting: if a later version stops honoring it, create fails, and leaves nothing behind.
The default webhooks are not listed here, and their IDs are refused: see gitclt webhook default.`,
		args: plainArgs, split: plainSplit(webhooks.System())},
	{name: "default", short: "Templates copied into each repository created afterwards (Forgejo only, needs admin rights)",
		long: `Existing repositories are not affected, and the copies are independent repository webhooks: editing or deleting a default
webhook leaves them as they are. Forgejo's API cannot list default webhooks: create shows the ID to keep, and the web interface lists them.`,
		noList: true, args: plainArgs, split: plainSplit(webhooks.Default())},
	{name: "repo", aliases: []string{"repository"}, short: "A repository's webhooks", prefix: "[<REPO>] ", ping: true,
		args: repoArgs, split: repoSplit},
	{name: "org", aliases: []string{"organization"}, short: "An organization's webhooks (a workspace on Bitbucket)", prefix: "<ORG> ", ping: true,
		args: orgArgs, split: orgSplit},
}

const hookCreateHelp = `Without --event, the webhook subscribes to push events. --event all subscribes to everything: GitHub's wildcard, or the events the library knows elsewhere.
Forgejo's names (ex: workflow_run_failure) also work; workflow_run and workflow_job stand for all of their statuses, on GitHub and on Forgejo 17 and later.
A name Forgejo does not know is dropped by the server: the command then says so, and the webhook exists without it.
The secret signs the deliveries; like the authorization header, it is never given on the command line: --secret asks for it, --secret-file reads it
from a file only you can access. Neither can be read back.
--type chooses the format of the deliveries (Forgejo only; default: forgejo). --type slack also serves Mattermost, and needs --slack-channel.`

const hookEditHelp = `Only the flags passed are changed; Booleans are revoked with =false, and "" clears a text field.
--event replaces the whole list. Forgejo's API cannot change the package, action_run_*, workflow_run_* and workflow_job_* events of an existing webhook:
keep those as they are, or the command refuses, and changes nothing; to change them, create a new webhook and delete this one.
Forgejo's API cannot change a webhook's secret, nor remove its authorization header: the command refuses these too.
A webhook's type cannot be changed; the Slack flags only change the settings passed.`

func hookCommands(s hookScope) *cobra.Command {
	parent := &cobra.Command{Use: s.name, Aliases: s.aliases, Short: s.short, Long: s.long}
	example := func(rest string) string { return "  gitclt webhook " + s.name + " " + rest }

	// Hidden where the API cannot list: it is there to say so, as it is the first thing anyone would try
	list := &cobra.Command{
		Use:     "list" + trimmedSuffix(s.prefix),
		Aliases: []string{"ls"},
		Hidden:  s.noList,
		Example: example("list [-c CONNECTION] [--json]" + trimmedSuffix(s.prefix)),
		Short:   "Lists the webhooks; -q outputs only their IDs",
		Args:    s.args(0, false),
		Run: func(cmd *cobra.Command, args []string) {
			t, _ := s.split(args, 0, false)
			if _, e := webhooks.ListHooks(t, true); e != nil {
				fmt.Println(e.Error())
			}
		},
	}
	list.Flags().BoolVarP(&webhooks.JSONoutput, "json", "j", false, "output as JSON")
	parent.AddCommand(list)

	get := &cobra.Command{
		Use:     "get " + s.prefix + "<ID>",
		Aliases: []string{"info"},
		Example: example("get [-c CONNECTION] [--json] " + s.prefix + "<ID>"),
		Short:   "Shows a webhook, with all of its events",
		Args:    s.args(1, false),
		Run: func(cmd *cobra.Command, args []string) {
			t, rest := s.split(args, 1, false)
			if _, e := webhooks.GetHook(t, rest[0], true); e != nil {
				fmt.Println(e.Error())
			}
		},
	}
	get.Flags().BoolVarP(&webhooks.JSONoutput, "json", "j", false, "output as JSON")

	create := &cobra.Command{
		Use:     "create " + s.prefix + "<URL>",
		Aliases: []string{"add"},
		Example: example("create [-c CONNECTION] [flags] "+s.prefix+"<URL>\n") +
			example("create -e push,release --secret-file ~/.hook-secret -b main "+s.prefix+"https://ci.example.com/hook"),
		Short: "Creates a webhook delivering to <URL>; -q outputs only its ID",
		Long:  hookCreateHelp,
		Args:  s.args(1, false),
		Run: func(cmd *cobra.Command, args []string) {
			t, rest := s.split(args, 1, false)
			if e := webhooks.CreateHook(t, rest[0], cmd.Flags().Changed); e != nil {
				fmt.Println(e.Error())
			}
		},
	}
	hookFlags(create)
	// The type is set once and for all: only create has the flag
	create.Flags().StringVarP(&webhooks.HookType, "type", "t", "", "format of the deliveries (Forgejo only; default: forgejo)")
	_ = create.RegisterFlagCompletionFunc("type", completeHookType)

	edit := &cobra.Command{
		Use:     "edit " + s.prefix + "<ID>",
		Aliases: []string{"modify", "update"},
		Example: example("edit [-c CONNECTION] [flags] "+s.prefix+"<ID>\n") +
			example("edit --active=false "+s.prefix+"42"),
		Short: "Edits a webhook; only the flags passed are changed",
		Long:  hookEditHelp,
		Args:  s.args(1, false),
		Run: func(cmd *cobra.Command, args []string) {
			t, rest := s.split(args, 1, false)
			if e := webhooks.EditHook(t, rest[0], cmd.Flags().Changed); e != nil {
				fmt.Println(e.Error())
			}
		},
	}
	hookFlags(edit)
	edit.Flags().StringVarP(&webhooks.HookURL, "url", "u", "", "URL to deliver to")

	del := &cobra.Command{
		Use:     "delete " + s.prefix + "<ID>...",
		Aliases: []string{"rm", "remove"},
		Example: example("delete [-c CONNECTION] [--yes] " + s.prefix + "<ID>..."),
		Short:   "Deletes webhooks, after confirmation",
		Long: `Every webhook is checked before asking, and the question shows their URLs.
The run stops at the first failure; the webhooks deleted before it stay deleted.`,
		Args: s.args(1, true),
		Run: func(cmd *cobra.Command, args []string) {
			t, rest := s.split(args, 1, true)
			if e := webhooks.DeleteHooks(t, rest); e != nil {
				fmt.Println(e.Error())
			}
		},
	}
	del.Flags().BoolVarP(&webhooks.AssumeYes, "yes", "y", false, "do not ask for confirmation before deleting")

	parent.AddCommand(get, create, edit, del)

	if s.ping {
		ping := &cobra.Command{
			Use:     "ping " + s.prefix + "<ID>",
			Aliases: []string{"test"},
			Example: example("ping [-c CONNECTION] " + s.prefix + "<ID>"),
			Short:   "Sends a test delivery (not on Bitbucket; on Forgejo, repositories only, as a push event)",
			Args:    s.args(1, false),
			Run: func(cmd *cobra.Command, args []string) {
				t, rest := s.split(args, 1, false)
				if e := webhooks.PingHook(t, rest[0]); e != nil {
					fmt.Println(e.Error())
				}
			},
		}
		parent.AddCommand(ping)
	}
	return parent
}

// list has nothing after what precedes the other arguments, so the space moves in front
func trimmedSuffix(prefix string) string {
	if prefix == "" {
		return ""
	}
	return " " + prefix[:len(prefix)-1]
}

func hookFlags(cmd *cobra.Command) {
	cmd.Flags().StringSliceVarP(&webhooks.HookEvents, "event", "e", nil, "event to subscribe to (repeatable, or comma-separated); all: everything")
	cmd.Flags().StringVarP(&webhooks.HookContentType, "content-type", "T", "", "payload encoding: json (the default) or form")
	cmd.Flags().StringVarP(&webhooks.HookBranchFilter, "branch-filter", "b", "", "limit push and branch events to matching branches, as a glob such as {main,release/*} (Forgejo only)")
	cmd.Flags().BoolVarP(&webhooks.HookActive, "active", "a", true, "the webhook delivers; =false keeps it without deliveries")
	cmd.Flags().BoolVar(&webhooks.HookInsecureSSL, "insecure-ssl", false, "do not check the target's TLS certificate: a third party could read or alter deliveries (GitHub, Bitbucket)")
	cmd.Flags().BoolVarP(&webhooks.HookPromptSecret, "secret", "s", false, "prompt for the secret that signs deliveries")
	cmd.Flags().StringVar(&webhooks.HookSecretFile, "secret-file", "", "read the secret that signs deliveries from this file, which only you can access")
	cmd.Flags().BoolVar(&webhooks.HookPromptAuth, "auth-header", false, "prompt for an authorization header sent with deliveries (Forgejo only)")
	cmd.Flags().StringVar(&webhooks.HookAuthFile, "auth-header-file", "", "read the authorization header from this file, which only you can access (Forgejo only)")
	cmd.Flags().StringVar(&webhooks.HookSlackChannel, "slack-channel", "", "channel of a slack webhook (also Mattermost)")
	cmd.Flags().StringVar(&webhooks.HookSlackUsername, "slack-username", "", "user name shown by a slack webhook")
	cmd.Flags().StringVar(&webhooks.HookSlackIconURL, "slack-icon-url", "", "icon shown by a slack webhook")
	cmd.Flags().StringVar(&webhooks.HookSlackColor, "slack-color", "", "color of a slack webhook's messages")
	cmd.MarkFlagsMutuallyExclusive("secret", "secret-file")
	cmd.MarkFlagsMutuallyExclusive("auth-header", "auth-header-file")
	_ = cmd.RegisterFlagCompletionFunc("content-type", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"json", "form"}, cobra.ShellCompDirectiveNoFileComp
	})
}

const hookIDHelp = `An ID is never taken for <REPO>, so several webhooks need no . (gitclt webhook repo delete 12 13);
a repository with a numeric name cannot be designated by this command.`

func init() {
	for _, s := range hookScopes {
		sub := hookCommands(s)
		webhookCmd.AddCommand(sub)
		switch s.name {
		case "repo":
			sub.PersistentFlags().StringVarP(&webhooks.OwnerOrg, "owner", "o", "", ownerFlagHelp)
			addHelp(sub, repoInferenceHelp)
			for _, c := range sub.Commands() {
				if c.Name() == "delete" {
					addHelp(c, hookIDHelp)
				}
			}
		case "system", "default":
			addHelp(sub, s.long)
		}
	}
}

func completeHookType(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{"forgejo", "gitea", "gogs", "slack", "discord", "dingtalk", "msteams", "feishu", "wechatwork"}, cobra.ShellCompDirectiveNoFileComp
}
