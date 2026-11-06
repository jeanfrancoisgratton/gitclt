// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repo_mgmt/webhooks/create.go
// Original timestamp : 2026.11.06 09:00:00

package webhooks

import (
	"context"
	"fmt"
	"gitclt/shared"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// CreateHook creates a webhook delivering to url; changed is cmd.Flags().Changed
// Without --event, it subscribes to push events
func CreateHook(t Target, url string, changed func(string) bool) *cerr.CustomError {
	ctx := context.Background()

	// Everything local comes first: nothing is created if a secret cannot be read
	opts, ce := createOptions(url, changed)
	if ce != nil {
		return ce
	}
	opened, ce := open(ctx, t)
	if ce != nil {
		return ce
	}

	hook, e := opened.store.Create(ctx, opts)
	if e != nil {
		return hookError("Unable to create the "+opened.noun, hook, opened, e)
	}

	if JSONoutput {
		return shared.PrintJSON(hook, "webhook")
	}
	if shared.QuietOutput {
		fmt.Println(hook.ID)
		return nil
	}
	fmt.Println(hftx.EnabledSign("Created " + hftx.Brown(opened.name(hook.ID))))
	if t.kind == kindDefault {
		fmt.Println(hftx.InfoSign("Keep this ID: the API cannot list default webhooks"))
	}
	return nil
}

func createOptions(url string, changed func(string) bool) (gol.CreateWebhookOptions, *cerr.CustomError) {
	opts := gol.CreateWebhookOptions{URL: url, Type: gol.HookType(strings.ToLower(HookType)), BranchFilter: HookBranchFilter, InsecureSSL: HookInsecureSSL,
		Events: []gol.HookEvent{gol.HookEventPush}}

	var ce *cerr.CustomError
	if opts.ContentType, ce = contentType(); ce != nil {
		return opts, ce
	}
	if len(HookEvents) > 0 {
		opts.Events = toEvents(HookEvents)
	}
	if changed("active") {
		opts.Active = gol.Ptr(HookActive)
	}
	if HookSlackChannel != "" || HookSlackUsername != "" || HookSlackIconURL != "" || HookSlackColor != "" {
		opts.Slack = &gol.SlackHookSettings{Channel: HookSlackChannel, Username: HookSlackUsername, IconURL: HookSlackIconURL, Color: HookSlackColor}
	}
	if opts.Secret, _, ce = readSecret(HookSecretFile, HookPromptSecret, "webhook secret"); ce != nil {
		return opts, ce
	}
	if opts.AuthorizationHeader, _, ce = readSecret(HookAuthFile, HookPromptAuth, "authorization header"); ce != nil {
		return opts, ce
	}
	return opts, nil
}
