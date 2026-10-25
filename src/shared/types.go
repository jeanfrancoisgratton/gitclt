// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/shared/types.go
// Original timestamp : 2026.10.18 19:28:57

package shared

import (
	"fmt"
	"strings"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

var QuietOutput = false

// The connection to use, from -c/--connection; empty means the default connection
var ConnectionName = ""

// DebugMode is set to true when running the app through the IDE step-by-step
var DebugMode = false

// VisibilityFlag is a pflag.Value restricted to the owner visibilities gol accepts; used by users, orgs, etc
type VisibilityFlag gol.OwnerVisibility

func (v *VisibilityFlag) String() string { return string(*v) }
func (v *VisibilityFlag) Type() string   { return "public|limited|private" }

func (v *VisibilityFlag) Set(s string) error {
	ov := gol.OwnerVisibility(strings.ToLower(s))
	if !ov.Valid() {
		return fmt.Errorf("must be one of public, limited or private")
	}
	*v = VisibilityFlag(ov)
	return nil
}
