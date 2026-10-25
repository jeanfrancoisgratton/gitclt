// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/connections/load_save.go
// Original timestamp: 2023/08/19 10:02

package connections

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
)

func connectionsFilePath() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "JFG", "gitclt", "connections.json")
}

func safeDecodeString(s string) string {
	// helperFunctions.DecodeString() panics if the input is not a valid
	// ciphertext produced by helperFunctions.EncodeString(). Since users may
	// still edit the connections file manually with cleartext secrets, fall back to
	// the original string. A cleartext value that happens to be valid base64 (ex: a
	// hex token) will not panic but decode to garbage, so we also reject non-printable results.
	if strings.TrimSpace(s) == "" {
		return ""
	}

	decoded := s
	func() {
		defer func() {
			if recover() != nil {
				decoded = s
			}
		}()
		decoded = hf.DecodeString(s, "")
	}()

	if !utf8.ValidString(decoded) {
		return s
	}
	for _, r := range decoded {
		if !unicode.IsPrint(r) {
			return s
		}
	}
	return decoded
}

func encodeString(s string) string {
	if s == "" {
		return ""
	}
	return hf.EncodeString(s, "")
}

// Load the JSON connections file in the user's config directory, and store it into a data type (struct)
// A missing file is not an error: we return an empty registry
func LoadConnectionsFile() (ConnectionsFileStruct, *cerr.CustomError) {
	payload := ConnectionsFileStruct{Auths: map[string]ConnectionStruct{}}

	jFile, err := os.ReadFile(connectionsFilePath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return payload, nil
		}
		return ConnectionsFileStruct{}, &cerr.CustomError{Title: "Error reading the connections file", Message: err.Error()}
	}
	if err = json.Unmarshal(jFile, &payload); err != nil {
		return ConnectionsFileStruct{}, &cerr.CustomError{Title: "Error parsing the connections file", Message: err.Error()}
	}
	if payload.Auths == nil {
		payload.Auths = map[string]ConnectionStruct{}
	}

	for name, c := range payload.Auths {
		c.Token = safeDecodeString(c.Token)
		c.Password = safeDecodeString(c.Password)
		payload.Auths[name] = c
	}
	return payload, nil
}

// Save the above structure into the JSON file in the user's config directory
// The whole registry is rewritten, so an updated entry replaces the existing one in-place
func (cf ConnectionsFileStruct) SaveConnectionsFile() *cerr.CustomError {
	out := ConnectionsFileStruct{Auths: make(map[string]ConnectionStruct, len(cf.Auths)), DefaultConnection: cf.DefaultConnection}
	for name, c := range cf.Auths {
		c.Token = encodeString(c.Token)
		c.Password = encodeString(c.Password)
		out.Auths[name] = c
	}

	// json/v2 does not sort map keys by default; we want a stable, diff-friendly file
	jStream, err := json.Marshal(out, jsontext.WithIndent("\t"), json.Deterministic(true))
	if err != nil {
		return &cerr.CustomError{Title: err.Error(), Fatality: cerr.Fatal}
	}

	// Write to a temp file first, then rename: we never leave a half-written registry behind
	rcFile := connectionsFilePath()
	tmpFile := rcFile + ".tmp"
	if err = os.WriteFile(tmpFile, jStream, 0600); err != nil {
		return &cerr.CustomError{Title: "Error writing the connections file", Message: err.Error()}
	}
	if err = os.Rename(tmpFile, rcFile); err != nil {
		_ = os.Remove(tmpFile)
		return &cerr.CustomError{Title: "Error writing the connections file", Message: err.Error()}
	}

	return nil
}
