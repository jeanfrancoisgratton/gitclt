// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/connections/active.go
// Original timestamp : 2026.10.18 14:00:00

package connections

import (
	"fmt"
	"net/url"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

// Fetches the named connection (an empty name means the default connection), and decides how to use it
// The token always takes precedence over the username/password pair
func ResolveConnection(name string) (ActiveConnection, *cerr.CustomError) {
	if name == "" {
		cf, err := LoadConnectionsFile()
		if err != nil {
			return ActiveConnection{}, err
		}
		name = cf.DefaultConnection
	}
	// GetConnection() already handles the "no default set", "unknown connection" and "invalid connection" cases
	c, err := GetConnection(name)
	if err != nil {
		return ActiveConnection{}, err
	}

	ac := ActiveConnection{Name: name, ServerType: c.ServerType, URL: strings.TrimRight(c.URL, "/"), Username: c.Username, Owner: c.Owner}
	if c.Token != "" {
		ac.Auth = AuthToken
		ac.Token = c.Token
	} else {
		// GitHub dropped password authentication on its REST API
		if c.ServerType == ServerTypeGithub {
			return ActiveConnection{}, &cerr.CustomError{Title: "Invalid connection",
				Message: fmt.Sprintf("Connection %s: GitHub requires an API token; fix it with: gitclt conn edit %s", name, name)}
		}
		ac.Auth = AuthBasic
		ac.Password = c.Password
	}

	// The API URL itself is derived by gitoperationslib (see forge.Connect()); we only make sure it can be
	if u, perr := url.Parse(ac.URL); perr != nil || u.Scheme == "" || u.Host == "" {
		return ActiveConnection{}, &cerr.CustomError{Title: "Invalid connection",
			Message: fmt.Sprintf("Connection %s: invalid server URL '%s'; fix it with: gitclt conn edit %s", name, c.URL, name)}
	}
	return ac, nil
}
