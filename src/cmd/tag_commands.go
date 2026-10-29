// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/tag_commands.go
// Original timestamp : 2026.10.29 22:30:00

package cmd

import (
	"fmt"
	"gitclt/release_mgmt/tags"

	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Tag commands, on the server only",
	Long: `These commands only work on the tags of the server (remote-only); the local working copy's tags are never
read nor changed: manage them with git (git tag, git push, git fetch).
A tag's release, files included, is managed with gitclt release.`,
}

var tagListCmd = &cobra.Command{
	Use:     "list <REPO>",
	Aliases: []string{"ls"},
	Example: "  gitclt tag list [-c CONNECTION] [-o OWNER] [-j] <REPO>",
	Short:   "Lists the tags of a repository on the server (remote-only), with their release if they have one",
	Long: `Remote-only: the local tags are not listed (git tag does that).
The tags are listed newest version first, with their commit, their release (Forgejo, GitHub) and annotation;
GitHub's tag list does not report the annotations: see tag get.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := tags.ListTags(args[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var tagGetCmd = &cobra.Command{
	Use:     "get <REPO> <TAG>",
	Aliases: []string{"info"},
	Example: "  gitclt tag get [-c CONNECTION] [-o OWNER] [-j] <REPO> <TAG>",
	Short:   "Shows a tag on the server (remote-only): its commit, annotation, release and source archives",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := tags.GetTag(args[0], args[1], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var tagCreateCmd = &cobra.Command{
	Use:     "create <REPO> <TAG>",
	Aliases: []string{"add"},
	Example: "  gitclt tag create [-c CONNECTION] [-o OWNER] [-t TARGET] [-m MESSAGE] <REPO> <TAG>\n" +
		"  gitclt tag create -t develop -m \"Version 1.2.0\" myrepo v1.2.0",
	Short: "Creates a tag on the server (remote-only)",
	Long: `Remote-only: the tag is created on the server, not in the local working copy (git fetch brings it there).
<TAG> is created on <TARGET> (-t, --target: a branch, tag or commit known to the server), or on the default branch's head.
-m, --message makes an annotated tag, otherwise the tag is lightweight; Bitbucket always annotates, with its own message.
An existing tag is refused.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := tags.CreateTag(args[0], args[1]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var tagDeleteCmd = &cobra.Command{
	Use:     "delete <REPO> <TAG>...",
	Aliases: []string{"rm", "del"},
	Example: "  gitclt tag delete [-c CONNECTION] [-o OWNER] [--with-release] [-y] <REPO> <TAG>...\n" +
		"  gitclt tag rm -y myrepo v1.2.0-rc1 v1.2.0-rc2\n" +
		"  gitclt tag rm --with-release myrepo v1.2.0-rc1",
	Short: "Deletes tags on the server (remote-only)",
	Long: `Remote-only: the local working copy's tags stay (git tag -d removes them).
Every tag is checked before asking: it must exist on the server. A tag with a release is refused, unless
--with-release is passed: the release, with its assets, is then deleted before the tag.
The run stops at the first failure; the tags deleted before it stay deleted.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := tags.DeleteTags(args[0], args[1:]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	tagCmd.AddCommand(tagListCmd, tagGetCmd, tagCreateCmd, tagDeleteCmd)

	tagCmd.PersistentFlags().StringVarP(&tags.OwnerOrg, "owner", "o", "", "user, organization or workspace that owns the repository (default: the connection's owner, else your username)")
	tagListCmd.Flags().BoolVarP(&tags.JSONoutput, "json", "j", false, "output as JSON")
	tagGetCmd.Flags().BoolVarP(&tags.JSONoutput, "json", "j", false, "output as JSON")
	tagCreateCmd.Flags().StringVarP(&tags.TagTarget, "target", "t", "", "branch, tag or commit to tag (default: the default branch)")
	tagCreateCmd.Flags().StringVarP(&tags.TagMessage, "message", "m", "", "annotation; makes an annotated tag (default: a lightweight tag)")
	tagDeleteCmd.Flags().BoolVarP(&tags.AssumeYes, "yes", "y", false, "confirm deletion")
	tagDeleteCmd.Flags().BoolVar(&tags.WithRelease, "with-release", false, "also delete the releases of the tags, with their assets")
}
