// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/helpers.go
// Original timestamp : 2026.10.25 20:31:32

package cmd

import "github.com/spf13/cobra"

// completeVisibility offers the values shared.VisibilityFlag accepts, for any --visibility flag
func completeVisibility(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{"public", "limited", "private"}, cobra.ShellCompDirectiveNoFileComp
}
