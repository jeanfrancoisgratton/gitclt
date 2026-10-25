// gitclt
// Tests the --visibility flag type

package user

import "testing"

func TestVisibilityFlag(t *testing.T) {
	var v VisibilityFlag
	if v.Type() != "public|limited|private" {
		t.Fatalf("Type = %q", v.Type())
	}
	if err := v.Set("Limited"); err != nil || v.String() != "limited" {
		t.Fatalf("got %q, %v", v.String(), err)
	}
	// A bad value is refused, and the previous one kept
	if err := v.Set("secret"); err == nil || v.String() != "limited" {
		t.Fatalf("got %q, %v", v.String(), err)
	}
}
