// gitclt
// Tests how the repositories' owner is decided: -o, the connection's owner, its username, then the server

package forge

import (
	"context"
	"fmt"
	"gitclt/connections"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestOwner_Precedence(t *testing.T) {
	for _, tc := range []struct {
		name, flag string
		ac         connections.ActiveConnection
		want       string
	}{
		{"flag wins", "acme", connections.ActiveConnection{Owner: "org", Username: "me"}, "acme"},
		{"connection owner", "", connections.ActiveConnection{Owner: "org", Username: "me"}, "org"},
		{"username", "", connections.ActiveConnection{Username: "me"}, "me"},
		{"bitbucket workspace", "", connections.ActiveConnection{ServerType: connections.ServerTypeBitBucket,
			Username: "me@example.com", Owner: "ws"}, "ws"},
	} {
		// None of these needs the server: a nil gitoperationslib client would panic if it were called
		c := &Client{Connection: tc.ac}
		if got, err := c.Owner(context.Background(), tc.flag); err != nil || got != tc.want {
			t.Errorf("%s: got %q, %v; want %q", tc.name, got, err, tc.want)
		}
	}
}

func TestOwner_Bitbucket_NoWorkspace(t *testing.T) {
	c := &Client{Connection: connections.ActiveConnection{Name: "bb", ServerType: connections.ServerTypeBitBucket,
		Username: "me@example.com"}}
	if _, err := c.Owner(context.Background(), ""); err == nil || err.Title != "Missing owner" {
		t.Fatalf("got %+v", err)
	}
}

// Neither an owner nor a usable username (none, or an e-mail): the server says whom the token belongs to
func TestOwner_AsksTheServer(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			fmt.Fprint(w, `{"version":"15.0.2"}`)
		case "/api/v1/user":
			calls.Add(1)
			fmt.Fprint(w, `{"id":1,"login":"tokenowner"}`)
		default:
			// Me() also reads emails, orgs, keys...; their absence does not matter here
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"not found"}`)
		}
	}))
	defer srv.Close()

	for _, username := range []string{"", "me@example.com"} {
		writeRegistry(t, `{"url":"`+srv.URL+`","token":"abc","username":"`+username+`","servertype":"forgejo"}`)
		c, ce := Connect(context.Background(), "")
		if ce != nil {
			t.Fatal(ce.Error())
		}
		if got, err := c.Owner(context.Background(), ""); err != nil || got != "tokenowner" {
			t.Errorf("username %q: got %q, %v", username, got, err)
		}
	}
	if calls.Load() == 0 {
		t.Fatal("the server was never asked")
	}
}
