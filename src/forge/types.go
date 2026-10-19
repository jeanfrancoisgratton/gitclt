// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/forge/types.go
// Original timestamp : 2026.10.19 14:00:00

package forge

import (
	"gitclt/conn_cfg"
	"time"

	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// Fast-fail: protects the connect / TLS handshake / response headers waits
var FastFailTimeout = 30 * time.Second

const userAgent = "gitclt"

// A gitoperationslib client, along with the connection it was opened on
// gol.Client is embedded: c.Repos(), c.Branches(), c.Releases()... are called directly on it
type Client struct {
	gol.Client
	Connection conn_cfg.ActiveConnection
}
