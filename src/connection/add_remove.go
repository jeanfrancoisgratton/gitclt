// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/connection/add_remove.go
// Original timestamp: 2023/09/15 08:23

package connection

import (
	"fmt"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// A connection needs a known server type, a URL, and some way to authenticate: a token, or a username/password pair

func (c ConnectionStruct) Validate() *cerr.CustomError {
	if c.ServerType == ServerTypeUnknown {
		return &cerr.CustomError{Title: "Invalid connection", Message: fmt.Sprintf("The server type is missing or unknown (supported: %s)", strings.Join(SupportedServerTypes(), ", "))}
	}
	if c.URL == "" {
		return &cerr.CustomError{Title: "Invalid connection", Message: "The server URL cannot be empty"}
	}
	if c.Token == "" && (c.Username == "" || c.Password == "") {
		return &cerr.CustomError{Title: "Invalid connection", Message: "You need either a token, or a username and password"}
	}
	return nil
}

func AddConnection(name string) *cerr.CustomError {
	cf, err := LoadConnectionsFile()
	if err != nil {
		return err
	}
	if _, ok := cf.Auths[name]; ok {
		return &cerr.CustomError{Title: "Connection already exists", Message: fmt.Sprintf("Connection %s already exists; use the edit command to modify it", name)}
	}

	var c ConnectionStruct
	// The first supported type is the default
	dfltType := SupportedServerTypes()[0]
	st := hf.GetStringValFromPrompt(fmt.Sprintf("Enter the server type (%s) [%s]: ", strings.Join(SupportedServerTypes(), ", "), dfltType))
	if st == "" {
		st = dfltType
	}
	if c.ServerType, err = ParseServerType(st); err != nil {
		return err
	}
	c.URL = hf.GetStringValFromPrompt("Enter the server URL (ex: https://myserver:myport): ")
	c.Token = hf.GetPassword("Enter your API token (leave empty if none): ", DebugMode)
	c.Username = hf.GetStringValFromPrompt("Enter your username: ")
	c.Password = hf.GetPassword("Enter your password: ", DebugMode)

	if err = c.Validate(); err != nil {
		return err
	}

	cf.Auths[name] = c
	// The first connection becomes the default one
	if cf.DefaultConnection == "" {
		cf.DefaultConnection = name
	}
	if err = cf.SaveConnectionsFile(); err != nil {
		return err
	}

	fmt.Println(hftx.EnabledSign(fmt.Sprintf("Connection %s added", name)))
	return nil
}

func RemoveConnection(name string) *cerr.CustomError {
	cf, err := LoadConnectionsFile()
	if err != nil {
		return err
	}
	if _, ok := cf.Auths[name]; !ok {
		return &cerr.CustomError{Title: "Unknown connection", Message: fmt.Sprintf("Connection %s does not exist", name)}
	}

	delete(cf.Auths, name)
	wasDefault := cf.DefaultConnection == name
	if wasDefault {
		cf.DefaultConnection = ""
	}
	if err = cf.SaveConnectionsFile(); err != nil {
		return err
	}

	fmt.Println(hftx.EnabledSign(fmt.Sprintf("Connection %s removed", name)))
	if wasDefault {
		fmt.Println(hftx.WarningSign("This was the default connection; no default connection is now set"))
	}
	return nil
}
