// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/03/01 15:41
// Original filename: src/conn_cfg/types.go

package conn_cfg

var DebugMode = false

// This is how the tool will know which connection (target forge) to use
// The registry lives in $HOME/.config/JFG/gitclt/connections.json, and looks a lot like $HOME/.docker/config.json

// The kind of forge a connection points to; the zero value means "not set"
// To support a new forge: add a constant (before serverTypeCount), and its name in serverTypeNames
type ServerType int

const (
	ServerTypeUnknown ServerType = iota
	ServerTypeForgejo
	ServerTypeGithub
	ServerTypeBitBucket
	serverTypeCount
)

// In the JSON file, the server type is stored by name, not by value: reordering the constants won't break existing files
var serverTypeNames = [serverTypeCount]string{
	ServerTypeUnknown:   "",
	ServerTypeForgejo:   "forgejo",
	ServerTypeGithub:    "github",
	ServerTypeBitBucket: "bitbucket",
}

type ConnectionStruct struct {
	URL        string     `json:"url"`
	Token      string     `json:"token,omitempty"`
	Username   string     `json:"username,omitempty"`
	Password   string     `json:"password,omitempty"`
	ServerType ServerType `json:"servertype"`
}

type ConnectionsFileStruct struct {
	Auths             map[string]ConnectionStruct `json:"auths"`
	DefaultConnection string                      `json:"default_connection"`
}

// How a request gets authenticated; decided once, when the connection is resolved
type AuthMethod int

const (
	AuthToken AuthMethod = iota
	AuthBasic
)

// Everything needed to open a forge client. forge.Connect() gets one of these from ResolveConnection(),
// so commands never have to look into the connections file, or decide by themselves how to authenticate
type ActiveConnection struct {
	Name       string // the connection's name in the registry
	ServerType ServerType
	URL        string // the server URL, as configured (without trailing slash)
	Auth       AuthMethod
	Token      string // set only when Auth == AuthToken
	Username   string // always kept when configured (it's useful for /users/{username}/... endpoints)
	Password   string // set only when Auth == AuthBasic
}
