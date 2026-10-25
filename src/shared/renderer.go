// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/shared/renderer.go
// Original timestamp : 2026.10.24 19:32:32

package shared

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/jeanfrancoisgratton/customError/v3"
	"github.com/jeanfrancoisgratton/helperFunctions/v5/prettyjson"
)

// PrintJSON encodes and pretty-prints any of the user types; what is only used in the error titles
func PrintJSON(v any, what string) *customError.CustomError {
	jStream, jerr := json.Marshal(v, jsontext.WithIndent("\t"))
	if jerr != nil {
		return &customError.CustomError{Title: "Unable to encode the " + what, Message: jerr.Error()}
	}
	if e := prettyjson.Print(jStream); e != nil {
		return &customError.CustomError{Title: "Unable to print the " + what, Message: e.Error()}
	}
	return nil
}
