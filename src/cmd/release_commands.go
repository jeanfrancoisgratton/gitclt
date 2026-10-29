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
	Use:     "upload",
	Aliases: []string{"up"},
	Example: "  gitclt release upload [-c CONNECTION] [-o OWNER] [--create [--draft] [--prerelease]] [--replace] <REPO> <TAG> <FILE>...\n" +
		"  gitclt rel up --create --replace myrepo v1.2.0 __redhat/*.rpm __debian/*.deb",
	Short: "Uploads files to the release of a tag",
	Long: `The files are attached to the release of <TAG>, under their base name.
Without --create, the release must already exist; with it, a missing release is created, titled after the tag.
The tag itself must already be on the server: push it first.
An asset name already taken is an error, unless --replace is passed: the old asset is then deleted before the upload.
Everything is checked before the first upload; the run stops at the first failed upload, and the files uploaded before it stay.`,
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
	releaseUploadCmd.Flags().BoolVar(&releases.CreateRelease, "create", false, "create the release if the tag has none")
	releaseUploadCmd.Flags().BoolVar(&releases.DraftRelease, "draft", false, "with --create: create it as a draft")
	releaseUploadCmd.Flags().BoolVar(&releases.Prerelease, "prerelease", false, "with --create: mark it as a prerelease")
	releaseUploadCmd.Flags().BoolVarP(&releases.ReplaceAssets, "replace", "r", false, "replace the assets that already have the same name")
}
