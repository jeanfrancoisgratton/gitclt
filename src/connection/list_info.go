// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/conn_cfg/list_info.go
// Original timestamp: 2023/09/13 16:01

package conn_cfg

import (
	"fmt"
	"os"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func secretMarker(s string) string {
	if s == "" {
		return ""
	}
	return hftx.Yellow("*ENCRYPTED*")
}

// Hand-edited connections are not validated on load (so they can still be fixed with edit or rm); we flag them here
func validityMarker(c ConnectionStruct) string {
	if err := c.Validate(); err != nil {
		return hftx.WarningSign(hftx.Yellow(err.Message))
	}
	return hftx.Green("OK")
}

func ListConnections() *cerr.CustomError {
	cf, err := LoadConnectionsFile()
	if err != nil {
		return err
	}

	fmt.Printf("Number of connections: %s\n", hftx.Green(fmt.Sprintf("%d", len(cf.Auths))))
	if len(cf.Auths) == 0 {
		return nil
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Connection", "Type", "Server URL", "Username", "Default", "Userpass", "Token", "Status"})

	for name, c := range cf.Auths {
		dflt := ""
		if name == cf.DefaultConnection {
			dflt = hftx.Green("*")
		}
		up := hftx.ThumbsUpSign("")
		tkn := hftx.ThumbsUpSign("")
		if c.Password == "" || c.Username == "" {
			up = hftx.ThumbsDownSign("")
		}
		if c.Token == "" {
			tkn = hftx.ThumbsDownSign("")
		}
		t.AppendRow([]any{hftx.Green(name), hftx.Green(c.ServerType.String()), hftx.Green(c.URL), hftx.Green(c.Username),
			dflt, up, tkn, validityMarker(c)})
	}
	t.SortBy([]table.SortBy{
		{Name: "Connection", Mode: table.Asc},
	})
	t.SetStyle(table.StyleBold)
	t.Style().Format.Header = text.FormatDefault
	t.Render()

	return nil
}

// Shows the details of the named connections; no names means the default connection
func ExplainConnections(names []string) *cerr.CustomError {
	cf, err := LoadConnectionsFile()
	if err != nil {
		return err
	}

	if len(names) == 0 {
		if cf.DefaultConnection == "" {
			return &cerr.CustomError{Title: "No connection specified", Message: "No connection name given, and no default connection is set"}
		}
		names = []string{cf.DefaultConnection}
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Connection", "Type", "Server URL", "Username", "Password", "Token", "Default", "Status"})

	for _, name := range names {
		c, ok := cf.Auths[name]
		if !ok {
			return &cerr.CustomError{Title: "Unknown connection", Message: fmt.Sprintf("Connection %s does not exist", name)}
		}
		dflt := ""
		if name == cf.DefaultConnection {
			dflt = hftx.Green("*")
		}
		t.AppendRow([]any{hftx.Green(name), hftx.Green(c.ServerType.String()), hftx.Green(c.URL), hftx.Green(c.Username),
			secretMarker(c.Password), secretMarker(c.Token), dflt, validityMarker(c)})
	}
	t.SortBy([]table.SortBy{
		{Name: "Connection", Mode: table.Asc},
	})
	t.SetStyle(table.StyleBold)
	t.Style().Format.Header = text.FormatDefault
	t.Render()

	return nil
}
