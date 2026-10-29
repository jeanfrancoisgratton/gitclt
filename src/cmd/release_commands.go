// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/release_commands.go
// Original timestamp : 2026.10.29 17:20:00

package cmd

import (
	"fmt"
	"gitclt/release_mgmt/releases"

	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:     "release",
	Aliases: []string{"rel"},
	Short:   "Release commands",
	Long: `Releases are the forge's own records attached to a tag; files (assets) are attached to releases, never to tags.
Forgejo and GitHub only: Bitbucket Cloud has no releases.`,
}

var releaseUploadCmd = &cobra.Command{
	Use:     "upload <REPO> <TAG> <FILE>...",
	Aliases: []string{"up"},
	Example: "  gitclt release upload [-c CONNECTION] [-o OWNER] [--create [--draft] [--prerelease]] [-r] <REPO> <TAG> <FILE>...\n" +
		"  gitclt release upload myrepo v1.2.0 pkg-1.2.0.x86_64.rpm\n" +
		"  gitclt rel up --create -r myrepo v1.2.0 __redhat/*.rpm __debian/*.deb __alpine/*.apk\n" +
		"  gitclt rel up --create --draft --prerelease -o myorg myrepo v2.0.0-rc1 dist/*",
	Short: "Uploads files to the release of a tag; Forgejo and GitHub only",
	Long: `<REPO> is the repository, <TAG> the tag whose release gets the files, and <FILE>... the local files to upload.
Each file becomes an asset of the release, named after its base name (dist/pkg.rpm is uploaded as pkg.rpm).
Files are attached to the release of a tag, never to the tag itself; Bitbucket Cloud has no releases.
Without --create, the release must already exist; --create creates a missing one, titled after the tag
(--draft and --prerelease only apply then). Either way, the tag must already be on the server: push it first.
An asset name already taken is an error, unless -r (--replace) is passed: the old asset is then deleted before the upload.
Everything (files, release, asset names) is checked before the first upload; the run stops at the first failed upload,
and the files uploaded before it stay.`,
	Args: cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if e := releases.UploadAssets(args[0], args[1], args[2:]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var releaseListCmd = &cobra.Command{
	Use:     "list <REPO>",
	Aliases: []string{"ls"},
	Example: "  gitclt release list [-c CONNECTION] [-o OWNER] [-j] <REPO>",
	Short:   "Lists the releases of a repository, drafts included; Forgejo and GitHub only",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := releases.ListReleases(args[0], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var releaseDeleteCmd = &cobra.Command{
	Use:     "delete <REPO> <TAG>...",
	Aliases: []string{"rm"},
	Example: "  gitclt release delete [-c CONNECTION] [-o OWNER] [-y] <REPO> <TAG>...\n" +
		"  gitclt rel rm -y myrepo v1.2.0-rc1 v1.2.0-rc2",
	Short: "Deletes the releases of tags, with their assets; Forgejo and GitHub only",
	Long: `<REPO> is the repository, and <TAG>... the tags whose releases are deleted, drafts included.
The release's assets go with it; the tags themselves stay on the server.
Every tag is checked before asking; the run stops at the first failed deletion, and the releases deleted before it stay deleted.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := releases.DeleteReleases(args[0], args[1:]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var releaseAssetsCmd = &cobra.Command{
	Use:     "list-assets <REPO> <TAG>",
	Aliases: []string{"lsa"},
	Example: "  gitclt release list-assets [-c CONNECTION] [-o OWNER] [-j] <REPO> <TAG>",
	Short:   "Lists the files attached to the release of a tag; Forgejo and GitHub only",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if _, e := releases.ListAssets(args[0], args[1], true); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var releaseDeleteAssetsCmd = &cobra.Command{
	Use:     "delete-assets <REPO> <TAG> <ASSET>...",
	Aliases: []string{"rma", "dla"},
	Example: "  gitclt release delete-assets [-c CONNECTION] [-o OWNER] [-y] <REPO> <TAG> <ASSET>...\n" +
		"  gitclt rel rma myrepo v1.2.0 pkg-1.2.0.x86_64.rpm pkg_1.2.0_amd64.deb",
	Short: "Deletes files attached to the release of a tag; Forgejo and GitHub only",
	Long: `<REPO> is the repository, <TAG> the tag whose release has the files, and <ASSET>... the asset names (see release assets).
Every name is checked before asking; the run stops at the first failed deletion, and the assets deleted before it stay deleted.`,
	Args: cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if e := releases.DeleteAssets(args[0], args[1], args[2:]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var releaseEditCmd = &cobra.Command{
	Use:     "edit <REPO> <TAG>",
	Aliases: []string{"modify", "update"},
	Example: "  gitclt release edit [-c CONNECTION] [-o OWNER] [flags] <REPO> <TAG>\n" +
		"  gitclt rel edit --title \"Version 1.2.0\" --notes-file docs/notes-1.2.0.md myrepo v1.2.0\n" +
		"  gitclt rel edit --prerelease=false --draft=false myrepo v2.0.0",
	Short: "Edits the release of a tag; only the flags passed are changed; Forgejo and GitHub only",
	Long: `<REPO> is the repository, and <TAG> the tag whose release is edited, drafts included.
Only the fields whose flag is passed are sent to the server; everything else is left unchanged.
Boolean flags can be revoked with =false: --draft=false publishes a draft (see release publish), and --draft turns
a published release back into a draft. An empty string clears a text field (ex: --notes "").
--notes and --notes-file both set the release notes (Markdown); pass one or the other.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := releases.EditRelease(args[0], args[1], cmd.Flags().Changed); e != nil {
			fmt.Println(e.Error())
		}
	},
}

var releasePublishCmd = &cobra.Command{
	Use:     "publish <REPO> <TAG>...",
	Aliases: []string{"pub"},
	Example: "  gitclt release publish [-c CONNECTION] [-o OWNER] <REPO> <TAG>...\n" +
		"  gitclt rel pub myrepo v1.2.0",
	Short: "Publishes the draft releases of tags; Forgejo and GitHub only",
	Long: `<REPO> is the repository, and <TAG>... the tags whose draft releases are published; releases already published are skipped.
Every tag is checked before the first publication: it must have a release, and be on the server (push it first).
The run stops at the first failure; the releases published before it stay published.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if e := releases.PublishReleases(args[0], args[1:]); e != nil {
			fmt.Println(e.Error())
		}
	},
}

func init() {
	releaseCmd.AddCommand(releaseUploadCmd, releaseListCmd, releaseDeleteCmd, releaseAssetsCmd, releaseDeleteAssetsCmd,
		releaseEditCmd, releasePublishCmd)

	releaseCmd.PersistentFlags().StringVarP(&releases.OwnerOrg, "owner", "o", "", "user, organization or workspace that owns the repository (default: the connection's owner, else your username)")
	releaseUploadCmd.Flags().BoolVar(&releases.CreateRelease, "create", false, "create the release if the tag has none (the tag must already be on the server)")
	releaseUploadCmd.Flags().BoolVar(&releases.DraftRelease, "draft", false, "with --create: create the release as a draft (not visible to the public until published)")
	releaseUploadCmd.Flags().BoolVar(&releases.Prerelease, "prerelease", false, "with --create: mark the release as a prerelease")
	releaseUploadCmd.Flags().BoolVarP(&releases.ReplaceAssets, "replace", "r", false, "replace the assets that already have the same name (default: refuse to upload)")
	releaseListCmd.Flags().BoolVarP(&releases.JSONoutput, "json", "j", false, "output as JSON")
	releaseAssetsCmd.Flags().BoolVarP(&releases.JSONoutput, "json", "j", false, "output as JSON")
	releaseDeleteCmd.Flags().BoolVarP(&releases.AssumeYes, "yes", "y", false, "confirm deletion")
	releaseDeleteAssetsCmd.Flags().BoolVarP(&releases.AssumeYes, "yes", "y", false, "confirm deletion")

	releaseEditCmd.Flags().StringVar(&releases.EditTitle, "title", "", "title of the release")
	releaseEditCmd.Flags().StringVar(&releases.EditNotes, "notes", "", "release notes (Markdown)")
	releaseEditCmd.Flags().StringVar(&releases.EditNotesFile, "notes-file", "", "read the release notes from this file")
	releaseEditCmd.Flags().BoolVar(&releases.EditDraft, "draft", false, "make the release a draft (--draft=false publishes it)")
	releaseEditCmd.Flags().BoolVar(&releases.EditPrerelease, "prerelease", false, "mark the release as a prerelease (--prerelease=false reverts it)")
	releaseEditCmd.MarkFlagsMutuallyExclusive("notes", "notes-file")
}
