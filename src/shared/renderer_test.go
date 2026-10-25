// gitclt
// Tests PrintJSON's output, and its error title

package shared

import (
	"strings"
	"testing"

	"gitclt/testutil"
)

func TestPrintJSON(t *testing.T) {
	v := struct {
		Login string `json:"login"`
		ID    int    `json:"id"`
	}{"jfg", 42}

	out := testutil.Run(t, func() {
		if err := PrintJSON(v, "user"); err != nil {
			t.Fatal(err.Error())
		}
	})
	if !strings.Contains(out, `"login"`) || !strings.Contains(out, `"jfg"`) || !strings.Contains(out, "42") {
		t.Fatalf("output = %q", out)
	}
}

func TestPrintJSON_Unencodable(t *testing.T) {
	err := PrintJSON(make(chan int), "user list")
	if err == nil || err.Title != "Unable to encode the user list" {
		t.Fatalf("got %+v", err)
	}
}
