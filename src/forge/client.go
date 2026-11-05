// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/forge/client.go
// Original timestamp : 2026.10.19 14:00:00

package forge

import (
	"context"
	"errors"
	"fmt"
	"gitclt/connections"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	"github.com/jeanfrancoisgratton/gitoperationslib/bitbucket"
	"github.com/jeanfrancoisgratton/gitoperationslib/forgejo"
	"github.com/jeanfrancoisgratton/gitoperationslib/github"
)

// Resolves the named connection (an empty name means the default connection), and opens a gitoperationslib client on it
// On Forgejo, this already contacts the server: it checks that its version is supported (GitHub and Bitbucket make no request here)
func Connect(ctx context.Context, connName string) (*Client, *cerr.CustomError) {
	ac, err := connections.ResolveConnection(connName)
	if err != nil {
		return nil, err
	}

	var gc gol.Client
	var gerr error
	switch ac.ServerType {
	case connections.ServerTypeForgejo:
		// A token needs no two-factor code, whether the account has two-factor authentication or not
		if ac.Auth == connections.AuthBasic {
			var ce *cerr.CustomError
			if gc, ce = connectForgejoPassword(ctx, ac, ac.Username, ac.Password); ce != nil {
				return nil, ce
			}
			break
		}
		gc, gerr = forgejo.New(ctx, ac.URL, forgejo.Token(ac.Token), forgejo.WithHTTPClient(httpClient()), forgejo.WithUserAgent(userAgent))
	case connections.ServerTypeGithub:
		// ResolveConnection() already refused GitHub connections without a token
		opts := []github.Option{github.WithHTTPClient(httpClient()), github.WithUserAgent(userAgent)}
		if !isGithubDotCom(ac.URL) {
			// GitHub Enterprise Server; the library appends /api/v3/
			opts = append(opts, github.WithEnterpriseURL(ac.URL, ""))
		}
		gc, gerr = github.New(ac.Token, opts...)
	case connections.ServerTypeBitBucket:
		// Bitbucket Cloud only (no Data Center)
		// Access tokens (workspace / project / repository) are bearer tokens, but Atlassian user API tokens (ATATT...)
		// need the Atlassian account e-mail, which we take from the connection's username
		creds := bitbucket.Token(ac.Token)
		switch {
		case ac.Auth == connections.AuthBasic:
			creds = bitbucket.BasicAuth(ac.Username, ac.Password)
		case isAtlassianAPIToken(ac.Token):
			if ac.Username == "" {
				return nil, &cerr.CustomError{Title: "Invalid connection",
					Message: fmt.Sprintf("Connection %s: an Atlassian API token needs your Atlassian account e-mail as username; fix it with: gitclt conn edit %s", ac.Name, ac.Name)}
			}
			creds = bitbucket.APIToken(ac.Username, ac.Token)
		}
		opts := []bitbucket.Option{bitbucket.WithHTTPClient(httpClient()), bitbucket.WithUserAgent(userAgent)}
		if !isBitbucketDotOrg(ac.URL) {
			// Any other host is taken as the API root itself (ex: a proxy, or https://host/2.0)
			opts = append(opts, bitbucket.WithBaseURL(ac.URL))
		}
		gc, gerr = bitbucket.New(creds, opts...)
	default:
		return nil, &cerr.CustomError{Title: "Invalid connection", Message: fmt.Sprintf("Connection %s: unsupported server type '%s'", ac.Name, ac.ServerType)}
	}
	if gerr != nil {
		return nil, Error(fmt.Sprintf("Unable to connect to %s", ac.Name), gerr)
	}
	return &Client{Client: gc, Connection: ac}, nil
}

// Opens a Forgejo client on an already resolved connection, with this username and password, whatever the connection's credentials are
// Forgejo refuses token authentication on a few endpoints (ex: creating or deleting access tokens)
func ConnectWithPassword(ctx context.Context, ac connections.ActiveConnection, username, password string) (*Client, *cerr.CustomError) {
	gc, ce := connectForgejoPassword(ctx, ac, username, password)
	if ce != nil {
		return nil, ce
	}
	return &Client{Client: gc, Connection: ac}, nil
}

// Converts a gitoperationslib error into our own error type; the HTTP status, if any, ends up in Code
func Error(title string, err error) *cerr.CustomError {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if isUnknownBitbucketUser(err) {
		msg += "; Bitbucket knows its users by display name, nickname, account ID or {UUID}, not by workspace nor e-mail: gitclt user whoami -x shows yours"
	}
	return &cerr.CustomError{Title: title, Message: msg, Code: gol.StatusCode(err)}
}

// The library has no sentinel for a name it could not resolve among a workspace's members: it is a 404 with this message
func isUnknownBitbucketUser(err error) bool {
	var ae *gol.APIError
	return errors.As(err, &ae) && ae.Backend == gol.BackendBitbucket && ae.StatusCode == http.StatusNotFound &&
		strings.HasPrefix(ae.Message, "no member ")
}

// Returns the account-management interface (users, orgs, teams); only Forgejo implements it
// gol.Admin is not part of gol.Client, so it is not promoted through the embedding: we need a type assertion
func (c *Client) Admin() (gol.Admin, *cerr.CustomError) {
	adm, ok := c.Client.(gol.Admin)
	if !ok {
		return nil, &cerr.CustomError{Title: "Unsupported operation",
			Message: fmt.Sprintf("Connection %s: account administration is only available on Forgejo", c.Connection.Name)}
	}
	return adm, nil
}

// Returns the commit history interface; the three forges implement it, but it is not part of gol.Client either
func (c *Client) History() (gol.History, *cerr.CustomError) {
	h, ok := c.Client.(gol.History)
	if !ok {
		return nil, &cerr.CustomError{Title: "Unsupported operation",
			Message: fmt.Sprintf("Connection %s: the commit history is not available on this forge", c.Connection.Name)}
	}
	return h, nil
}

// Returns the interface managing the authenticated user's own account; it is not part of gol.Client either
func (c *Client) Account() (gol.Account, *cerr.CustomError) {
	a, ok := c.Client.(gol.Account)
	if !ok {
		return nil, &cerr.CustomError{Title: "Unsupported operation",
			Message: fmt.Sprintf("Connection %s: account management is not available on this forge", c.Connection.Name)}
	}
	return a, nil
}

// github.com has its API on a separate host (api.github.com), which the library uses by default
func isGithubDotCom(serverURL string) bool {
	u, err := url.Parse(serverURL)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Hostname()) {
	case "github.com", "www.github.com", "api.github.com":
		return true
	}
	return false
}

// bitbucket.org has its API on a separate host (api.bitbucket.org/2.0/), which the library uses by default
func isBitbucketDotOrg(serverURL string) bool {
	u, err := url.Parse(serverURL)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Hostname()) {
	case "bitbucket.org", "www.bitbucket.org", "api.bitbucket.org":
		return true
	}
	return false
}

// Atlassian user API tokens start with ATATT; access tokens start with ATCTT
func isAtlassianAPIToken(token string) bool {
	return strings.HasPrefix(token, "ATATT")
}

// Fast-fail timeouts on connect / TLS handshake / response headers
// No http.Client.Timeout: it would also kill long transfers (ex: release assets); commands use contexts instead
func httpClient() *http.Client {
	dialer := &net.Dialer{Timeout: FastFailTimeout, KeepAlive: 30 * time.Second}
	return &http.Client{Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   FastFailTimeout,
		ResponseHeaderTimeout: FastFailTimeout,
		IdleConnTimeout:       90 * time.Second,
	}}
}
