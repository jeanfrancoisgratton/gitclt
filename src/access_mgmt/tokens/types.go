// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/access_mgmt/tokens/types.go
// Original timestamp : 2026.11.04 10:00:00

package tokens

import gol "github.com/jeanfrancoisgratton/gitoperationslib"

var JSONoutput = false
var AssumeYes = false

// Designates a token by its ID instead of its name
var TokenID int64

// The current 2FA code, for the commands authenticating with a password
var OTP string

// CreateTokenOptions flags
var CreateTokenScopes []string
var CreateTokenRepos []string

// What create outputs as JSON: the library never serialises the secret, and it is the whole point of creating a token
type createdToken struct {
	gol.AccessToken
	Secret string `json:"secret"`
}
