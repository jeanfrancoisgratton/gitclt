// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/conn_cfg/servertype.go
// Original timestamp : 2026.10.18 13:00:00

package conn_cfg

import (
	"fmt"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
)

func (st ServerType) String() string {
	if st < 0 || st >= serverTypeCount {
		return ""
	}
	return serverTypeNames[st]
}

// The names of all supported server types (thus excluding ServerTypeUnknown)
func SupportedServerTypes() []string {
	names := make([]string, 0, serverTypeCount-1)
	for st := ServerTypeUnknown + 1; st < serverTypeCount; st++ {
		names = append(names, st.String())
	}
	return names
}

func ParseServerType(s string) (ServerType, *cerr.CustomError) {
	s = strings.ToLower(strings.TrimSpace(s))
	for st := ServerTypeUnknown + 1; st < serverTypeCount; st++ {
		if s == serverTypeNames[st] {
			return st, nil
		}
	}
	return ServerTypeUnknown, &cerr.CustomError{Title: "Invalid server type",
		Message: fmt.Sprintf("Unknown server type '%s'; supported types are: %s", s, strings.Join(SupportedServerTypes(), ", "))}
}

// json/v2 uses these to (un)marshal the server type as a string
func (st ServerType) MarshalText() ([]byte, error) {
	return []byte(st.String()), nil
}

// An unknown or missing server type is not a parsing error: like any other hand-edited mistake, it
// gets flagged by Validate(), so the connection can still be fixed with edit or rm
func (st *ServerType) UnmarshalText(b []byte) error {
	*st, _ = ParseServerType(string(b))
	return nil
}
