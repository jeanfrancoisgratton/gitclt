// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repo_mgmt/webhooks/helpers.go
// Original timestamp : 2026.11.06 09:00:00

package webhooks

import (
	"context"
	"fmt"
	"gitclt/access_mgmt/users"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"os"
	"strconv"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v5"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"golang.org/x/term"
)

// hookStore is what every kind of webhook can do; the library's interfaces for each kind all fit it
type hookStore interface {
	Get(ctx context.Context, id int64) (*gol.Webhook, error)
	Create(ctx context.Context, opts gol.CreateWebhookOptions) (*gol.Webhook, error)
	Edit(ctx context.Context, id int64, opts gol.EditWebhookOptions) (*gol.Webhook, error)
	Delete(ctx context.Context, id int64) error
}

// Default webhooks cannot be listed: the API has no call for it
type hookLister interface {
	List(ctx context.Context, opts gol.ListOptions) (gol.Page[gol.Webhook], error)
}

// Only repository and organization webhooks have test deliveries
type hookPinger interface {
	Ping(ctx context.Context, id int64) error
}

// scopedHooks gives repository and organization webhooks the same shape as the other kinds
type scopedHooks struct {
	w     gol.Webhooks
	scope gol.HookScope
}

func (s scopedHooks) List(ctx context.Context, opts gol.ListOptions) (gol.Page[gol.Webhook], error) {
	return s.w.List(ctx, s.scope, opts)
}
func (s scopedHooks) Get(ctx context.Context, id int64) (*gol.Webhook, error) {
	return s.w.Get(ctx, s.scope, id)
}
func (s scopedHooks) Create(ctx context.Context, opts gol.CreateWebhookOptions) (*gol.Webhook, error) {
	return s.w.Create(ctx, s.scope, opts)
}
func (s scopedHooks) Edit(ctx context.Context, id int64, opts gol.EditWebhookOptions) (*gol.Webhook, error) {
	return s.w.Edit(ctx, s.scope, id, opts)
}
func (s scopedHooks) Delete(ctx context.Context, id int64) error {
	return s.w.Delete(ctx, s.scope, id)
}
func (s scopedHooks) Ping(ctx context.Context, id int64) error { return s.w.Ping(ctx, s.scope, id) }

// target is an opened Target: the store of its webhooks, and how to name them in messages
type target struct {
	store hookStore
	noun  string // "system webhook"
	where string // " of owner/repo"
}

func (t target) name(id int64) string { return fmt.Sprintf("%s %d%s", t.noun, id, t.where) }

func open(ctx context.Context, t Target) (target, *cerr.CustomError) {
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return target{}, ce
	}
	switch t.kind {
	case kindUser:
		if ce := c.Require(gol.CapUserWebhooks, "user webhooks only exist on Forgejo"); ce != nil {
			return target{}, ce
		}
		return target{store: c.Account().Webhooks(), noun: "user webhook"}, nil
	case kindSystem, kindDefault:
		if ce := c.Require(gol.CapSystemWebhooks, "system and default webhooks only exist on Forgejo"); ce != nil {
			return target{}, ce
		}
		h := c.Site()
		if t.kind == kindSystem {
			return target{store: h.SystemWebhooks(), noun: "system webhook"}, nil
		}
		return target{store: h.DefaultWebhooks(), noun: "default webhook"}, nil
	case kindRepo:
		owner, repo, ce := c.Target(ctx, OwnerOrg, t.repo)
		if ce != nil {
			return target{}, ce
		}
		return target{store: scopedHooks{c.Webhooks(), gol.RepoHooks(owner, repo)}, noun: "webhook", where: " of " + owner + "/" + repo}, nil
	default:
		return target{store: scopedHooks{c.Webhooks(), gol.OrgHooks(t.org)}, noun: "webhook", where: " of " + t.org}, nil
	}
}

// parseIDs turns arguments into webhook IDs, dropping repeated ones
func parseIDs(args []string) ([]int64, *cerr.CustomError) {
	ids := make([]int64, 0, len(args))
	seen := map[int64]bool{}
	for _, a := range args {
		id, e := strconv.ParseInt(a, 10, 64)
		if e != nil || id <= 0 {
			return nil, &cerr.CustomError{Title: "Invalid webhook ID", Message: a + " is not a webhook ID"}
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// The library may hand back a webhook along with its error: it exists, even though something went wrong
// (ex: Forgejo drops the event names it does not know)
func hookError(title string, hook *gol.Webhook, t target, e error) *cerr.CustomError {
	ce := forge.Error(title, e)
	if hook != nil {
		ce.Message += fmt.Sprintf("; %s exists nevertheless: check it with get, and fix it with edit or delete", t.name(hook.ID))
	}
	return ce
}

// A secret never comes from the command line, where any local user could read it: it is prompted for, or read from a
// file that only its owner can read
func readSecret(file string, prompt bool, what string) (string, bool, *cerr.CustomError) {
	switch {
	case file != "":
		value, e := users.ReadPasswordFile(file)
		if e != nil {
			return "", false, &cerr.CustomError{Title: "Unable to read the " + what, Message: e.Error()}
		}
		return value, true, nil
	case prompt:
		// GetPassword panics without a terminal, unless in debug mode
		if !shared.DebugMode && !term.IsTerminal(int(os.Stdin.Fd())) {
			return "", false, &cerr.CustomError{Title: "Unable to read the " + what,
				Message: "it cannot be asked for without a terminal; read it from a file instead"}
		}
		value := hf.GetPassword("Enter the "+what+": ", shared.DebugMode)
		if value == "" {
			return "", false, &cerr.CustomError{Title: "Unable to read the " + what, Message: "nothing was entered"}
		}
		return value, true, nil
	}
	return "", false, nil
}

func toEvents(names []string) []gol.HookEvent {
	events := make([]gol.HookEvent, 0, len(names))
	for _, n := range names {
		if strings.EqualFold(n, "all") {
			events = append(events, gol.HookEventAll)
			continue
		}
		events = append(events, gol.HookEvent(n))
	}
	return events
}

func contentType() (gol.HookContentType, *cerr.CustomError) {
	switch ct := gol.HookContentType(HookContentType); ct {
	case "", gol.HookContentJSON, gol.HookContentForm:
		return ct, nil
	}
	return "", &cerr.CustomError{Title: "Invalid content type", Message: HookContentType + " must be json or form", Code: http.StatusBadRequest}
}

// Deletion cannot be undone; ask unless --yes was passed
func confirm(question string) bool {
	if AssumeYes {
		return true
	}
	a := strings.ToLower(strings.TrimSpace(hf.GetStringValFromPrompt(hftx.WarningSign(question))))
	return a == "y" || a == "yes"
}
