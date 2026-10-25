// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/connection/resolve.go
// Original timestamp : 2026.10.18 12:30:00

package connection

import (
	"fmt"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

// Returns the named connection, ready to be used; an empty name means the default connection
// This is where a hand-edited, invalid connection gets caught

func GetConnection(name string) (ConnectionStruct, *cerr.CustomError) {
	cf, err := LoadConnectionsFile()
	if err != nil {
		return ConnectionStruct{}, err
	}

	if name == "" {
		if cf.DefaultConnection == "" {
			return ConnectionStruct{}, &cerr.CustomError{Title: "No connection specified", Message: "No connection name given, and no default connection is set"}
		}
		name = cf.DefaultConnection
	}

	c, ok := cf.Auths[name]
	if !ok {
		return ConnectionStruct{}, &cerr.CustomError{Title: "Unknown connection", Message: fmt.Sprintf("Connection %s does not exist", name)}
	}
	if err = c.Validate(); err != nil {
		err.Message = fmt.Sprintf("Connection %s: %s; fix it with: gitclt conn edit %s", name, err.Message, name)
		return ConnectionStruct{}, err
	}
	return c, nil
}
