// gitclt
// Tests conn list / conn info: secrets are never shown, invalid connections are flagged

package connections

import (
	"strings"
	"testing"

	"gitclt/testutil"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

const listRegistry = `{"auths":{
	"zulu":{"url":"https://zulu","token":"zulu-token","servertype":"github"},
	"alpha":{"url":"https://alpha","username":"me","password":"alpha-pw","servertype":"forgejo"},
	"broken":{"url":"","token":"t","servertype":"forgejo"}},"default_connection":"zulu"}`

func TestListConnections(t *testing.T) {
	testutil.WriteRegistry(t, listRegistry)

	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = ListConnections() })
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, want := range []string{"Number of connections: 3", "https://alpha", "https://zulu", "The server URL cannot be empty"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "zulu-token") || strings.Contains(out, "alpha-pw") {
		t.Errorf("a secret is shown:\n%s", out)
	}
	if strings.Index(out, "https://alpha") > strings.Index(out, "https://zulu") {
		t.Errorf("not sorted by name:\n%s", out)
	}
}

func TestListConnections_Empty(t *testing.T) {
	testutil.WriteRegistry(t, "")
	out := testutil.Run(t, func() { _ = ListConnections() })
	if strings.TrimSpace(out) != "Number of connections: 0" {
		t.Fatalf("output = %q", out)
	}
}

func TestExplainConnections(t *testing.T) {
	testutil.WriteRegistry(t, listRegistry)

	// No name: the default connection only
	var err *cerr.CustomError
	out := testutil.Run(t, func() { err = ExplainConnections(nil) })
	if err != nil || !strings.Contains(out, "https://zulu") || strings.Contains(out, "https://alpha") {
		t.Fatalf("default: %v\n%s", err, out)
	}
	if !strings.Contains(out, "*ENCRYPTED*") || strings.Contains(out, "zulu-token") {
		t.Fatalf("the secret is not masked:\n%s", out)
	}

	out = testutil.Run(t, func() { err = ExplainConnections([]string{"alpha", "broken"}) })
	if err != nil || !strings.Contains(out, "https://alpha") || !strings.Contains(out, "The server URL cannot be empty") {
		t.Fatalf("named: %v\n%s", err, out)
	}

	if err = ExplainConnections([]string{"nosuch"}); err == nil || err.Title != "Unknown connection" {
		t.Fatalf("unknown: got %+v", err)
	}
}

func TestExplainConnections_NoDefault(t *testing.T) {
	testutil.WriteRegistry(t, `{"auths":{}}`)
	if err := ExplainConnections(nil); err == nil || err.Title != "No connection specified" {
		t.Fatalf("got %+v", err)
	}
}

func TestListInfo_BrokenFile(t *testing.T) {
	testutil.WriteRegistry(t, `{`)
	if err := ListConnections(); err == nil {
		t.Error("list: expected an error")
	}
	if err := ExplainConnections(nil); err == nil {
		t.Error("info: expected an error")
	}
}
