// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/conn_cfg_commands.go
// Original timestamp : 2026.10.18 11:04:26

package cmd

import (
	"fmt"
	"gitclt/connection"

	"github.com/spf13/cobra"
)

var conn_cfgCmd = &cobra.Command{
	Use:     "conn",
	Aliases: []string{"connection"},
	Example: "gitclt conn list\n" +
		"  gitclt conn add CONNECTION\n" +
		"  gitclt conn info [CONNECTION...]\n" +
		"  gitclt conn edit CONNECTION\n" +
		"  gitclt conn rm CONNECTION\n" +
		"  gitclt conn default [CONNECTION]",
	Short: "Connections registry commands",
	Long:  `Manages the connections registry, $HOME/.config/JFG/gitclt/connections.json`,
}

var conn_cfgListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Example: "gitclt conn list",
	Short:   "Lists all connections in the registry",
	Run: func(cmd *cobra.Command, args []string) {
		if err := connection.ListConnections(); err != nil {
			fmt.Println(err.Error())
		}
	},
}

var conn_cfgInfoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"explain"},
	Example: "gitclt conn info [CONNECTION1 CONNECTION2... CONNECTIONn]",
	Short:   "Prints the details of the given connections",
	Long:    `You can list as many connections as you wish; no connection means the default one`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := connection.ExplainConnections(args); err != nil {
			fmt.Println(err.Error())
		}
	},
}

var conn_cfgAddCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"create"},
	Example: "gitclt conn add CONNECTION",
	Short:   "Adds CONNECTION to the registry",
	Long:    `The first connection added to the registry becomes the default connection`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := connection.AddConnection(args[0]); err != nil {
			fmt.Println(err.Error())
		}
	},
}

var conn_cfgRmCmd = &cobra.Command{
	Use:     "rm",
	Aliases: []string{"remove"},
	Example: "gitclt conn rm CONNECTION",
	Short:   "Removes CONNECTION from the registry",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := connection.RemoveConnection(args[0]); err != nil {
			fmt.Println(err.Error())
		}
	},
}

var conn_cfgEditCmd = &cobra.Command{
	Use:     "edit",
	Aliases: []string{"update"},
	Example: "gitclt conn edit CONNECTION",
	Short:   "Edits CONNECTION in-place",
	Long:    `Pressing ENTER at a prompt keeps the current value`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := connection.EditConnection(args[0]); err != nil {
			fmt.Println(err.Error())
		}
	},
}

var conn_cfgDefaultCmd = &cobra.Command{
	Use:     "default",
	Example: "gitclt conn default [CONNECTION]",
	Short:   "Shows or sets the default connection",
	Long:    `Without argument, shows the default connection; otherwise, sets it to CONNECTION`,
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			if err := connection.ShowDefaultConnection(); err != nil {
				fmt.Println(err.Error())
			}
			return
		}
		if err := connection.SetDefaultConnection(args[0]); err != nil {
			fmt.Println(err.Error())
		}
	},
}

func init() {
	conn_cfgCmd.AddCommand(conn_cfgListCmd, conn_cfgInfoCmd, conn_cfgAddCmd, conn_cfgRmCmd,
		conn_cfgEditCmd, conn_cfgDefaultCmd)
}
