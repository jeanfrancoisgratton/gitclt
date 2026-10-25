// gitclt
// Tests GetConnection: default connection, unknown and hand-edited invalid connections

package connections

import (
	"strings"
	"testing"

	"gitclt/testutil"
)

const twoConnections = `{"auths":{
	"good":{"url":"https://good","token":"t","servertype":"forgejo"},
	"broken":{"url":"","token":"t","servertype":"forgejo"}},
	"default_connection":"good"}`

func TestGetConnection(t *testing.T) {
	testutil.WriteRegistry(t, twoConnections)

	for _, name := range []string{"", "good"} {
		if c, err := GetConnection(name); err != nil || c.URL != "https://good" {
			t.Errorf("%q: got %+v, %v", name, c, err)
		}
	}

	if _, err := GetConnection("nosuch"); err == nil || err.Title != "Unknown connection" {
		t.Errorf("unknown: got %+v", err)
	}

	// The validation message tells how to fix it
	if _, err := GetConnection("broken"); err == nil || !strings.Contains(err.Message, "gitclt conn edit broken") {
		t.Errorf("broken: got %+v", err)
	}
}

func TestGetConnection_NoDefault(t *testing.T) {
	testutil.WriteRegistry(t, `{"auths":{"good":{"url":"https://good","token":"t","servertype":"forgejo"}}}`)
	if _, err := GetConnection(""); err == nil || err.Title != "No connection specified" {
		t.Fatalf("got %+v", err)
	}
}

func TestGetConnection_BrokenFile(t *testing.T) {
	testutil.WriteRegistry(t, `{`)
	if _, err := GetConnection("good"); err == nil {
		t.Fatal("expected an error")
	}
}
