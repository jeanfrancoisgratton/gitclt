// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/conn_cfg/edit.go
// Original timestamp : 2026.10.18 12:00:00

package conn_cfg

import (
	"fmt"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// Prompts for a new value; an empty answer keeps the current one
func promptKeep(prompt, current string) string {
	if v := hf.GetStringValFromPrompt(fmt.Sprintf("%s [%s]: ", prompt, current)); v != "" {
		return v
	}
	return current
}

// Same, for secrets: the current value is never shown, and "-" clears it
func promptSecretKeep(prompt, current string) string {
	hint := "empty"
	if current != "" {
		hint = "set"
	}
	v := hf.GetPassword(fmt.Sprintf("%s [%s; ENTER keeps, '-' clears]: ", prompt, hint), DebugMode)
	switch v {
	case "":
		return current
	case "-":
		return ""
	default:
		return v
	}
}

// Edits the connection in-place: the entry is replaced, never duplicated
func EditConnection(name string) *cerr.CustomError {
	cf, err := LoadConnectionsFile()
	if err != nil {
		return err
	}
	c, ok := cf.Auths[name]
	if !ok {
		return &cerr.CustomError{Title: "Unknown connection", Message: fmt.Sprintf("Connection %s does not exist; use the add command to create it", name)}
	}

	st := promptKeep(fmt.Sprintf("Server type (%s)", strings.Join(SupportedServerTypes(), ", ")), c.ServerType.String())
	if c.ServerType, err = ParseServerType(st); err != nil {
		return err
	}
	c.URL = promptKeep("Server URL", c.URL)
	c.Token = promptSecretKeep("API token", c.Token)
	c.Username = promptKeep("Username", c.Username)
	c.Password = promptSecretKeep("Password", c.Password)

	if err = c.Validate(); err != nil {
		return err
	}

	cf.Auths[name] = c
	if err = cf.SaveConnectionsFile(); err != nil {
		return err
	}

	fmt.Println(hftx.EnabledSign(fmt.Sprintf("Connection %s updated", name)))
	return nil
}

func ShowDefaultConnection() *cerr.CustomError {
	cf, err := LoadConnectionsFile()
	if err != nil {
		return err
	}
	if cf.DefaultConnection == "" {
		fmt.Println(hftx.WarningSign("No default connection is set"))
	} else {
		fmt.Printf("Default connection: %s\n", hftx.Green(cf.DefaultConnection))
	}
	return nil
}

// The default connection must point to an existing member of the auths section
func SetDefaultConnection(name string) *cerr.CustomError {
	cf, err := LoadConnectionsFile()
	if err != nil {
		return err
	}
	if _, ok := cf.Auths[name]; !ok {
		return &cerr.CustomError{Title: "Unknown connection", Message: fmt.Sprintf("Connection %s does not exist", name)}
	}

	cf.DefaultConnection = name
	if err = cf.SaveConnectionsFile(); err != nil {
		return err
	}

	fmt.Println(hftx.EnabledSign(fmt.Sprintf("Default connection set to %s", name)))
	return nil
}
