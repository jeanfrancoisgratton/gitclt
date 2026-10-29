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

func init() {
	releaseCmd.AddCommand(releaseUploadCmd)

	releaseCmd.PersistentFlags().StringVarP(&releases.OwnerOrg, "owner", "o", "", "user, organization or workspace that owns the repository (default: the connection's owner, else your username)")
	releaseUploadCmd.Flags().BoolVar(&releases.CreateRelease, "create", false, "create the release if the tag has none (the tag must already be on the server)")
	releaseUploadCmd.Flags().BoolVar(&releases.DraftRelease, "draft", false, "with --create: create the release as a draft (not visible to the public until published)")
	releaseUploadCmd.Flags().BoolVar(&releases.Prerelease, "prerelease", false, "with --create: mark the release as a prerelease")
	releaseUploadCmd.Flags().BoolVarP(&releases.ReplaceAssets, "replace", "r", false, "replace the assets that already have the same name (default: refuse to upload)")
}
