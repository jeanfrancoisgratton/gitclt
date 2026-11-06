// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repo_mgmt/webhooks/types.go
// Original timestamp : 2026.10.26 08:14:52

package webhooks

var JSONoutput = false
var AssumeYes = false
var OwnerOrg = ""

// create and edit flags
var HookURL string
var HookType string
var HookContentType string
var HookEvents []string
var HookBranchFilter string
var HookActive bool
var HookInsecureSSL bool
var HookSecretFile string
var HookPromptSecret bool
var HookAuthFile string
var HookPromptAuth bool
var HookSlackChannel string
var HookSlackUsername string
var HookSlackIconURL string
var HookSlackColor string

type kind int

const (
	kindUser kind = iota
	kindSystem
	kindDefault
	kindRepo
	kindOrg
)

// Target is where webhooks live
type Target struct {
	kind kind
	repo string // kindRepo; empty means the one the current directory is a clone of
	org  string // kindOrg
}

func User() Target            { return Target{kind: kindUser} }
func System() Target          { return Target{kind: kindSystem} }
func Default() Target         { return Target{kind: kindDefault} }
func Repo(repo string) Target { return Target{kind: kindRepo, repo: repo} }
func Org(org string) Target   { return Target{kind: kindOrg, org: org} }
