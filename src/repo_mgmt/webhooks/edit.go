// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/repo_mgmt/webhooks/edit.go
// Original timestamp : 2026.11.06 09:00:00

package webhooks

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"slices"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

var editFlags = []string{"url", "content-type", "event", "branch-filter", "active", "insecure-ssl", "secret", "secret-file",
	"auth-header", "auth-header-file", "slack-channel", "slack-username", "slack-icon-url", "slack-color"}

var slackFlags = []string{"slack-channel", "slack-username", "slack-icon-url", "slack-color"}

// EditHook only sends the fields whose flag was explicitly passed; changed is cmd.Flags().Changed
func EditHook(t Target, arg string, changed func(string) bool) *cerr.CustomError {
	ctx := context.Background()

	ids, ce := parseIDs([]string{arg})
	if ce != nil {
		return ce
	}
	if !slices.ContainsFunc(editFlags, changed) {
		return &cerr.CustomError{Title: "Nothing to edit", Message: "no flag was provided; see gitclt webhook --help"}
	}
	opts, ce := editOptions(changed)
	if ce != nil {
		return ce
	}
	opened, ce := open(ctx, t)
	if ce != nil {
		return ce
	}

	// The library replaces all of the Slack settings: the flags not passed keep their current value
	if slices.ContainsFunc(slackFlags, changed) {
		cur, e := opened.store.Get(ctx, ids[0])
		if e != nil {
			return forge.Error("Unable to fetch "+opened.name(ids[0]), e)
		}
		if cur.Slack == nil {
			return &cerr.CustomError{Title: "Not a Slack webhook", Message: fmt.Sprintf("%s has no Slack settings to change", opened.name(ids[0]))}
		}
		opts.Slack = mergeSlack(*cur.Slack, changed)
	}

	hook, e := opened.store.Edit(ctx, ids[0], opts)
	if e != nil {
		return hookError("Unable to edit "+opened.name(ids[0]), hook, opened, e)
	}

	if JSONoutput {
		return shared.PrintJSON(hook, "webhook")
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Updated " + hftx.Brown(opened.name(ids[0]))))
	}
	return nil
}

func mergeSlack(s gol.SlackHookSettings, changed func(string) bool) *gol.SlackHookSettings {
	if changed("slack-channel") {
		s.Channel = HookSlackChannel
	}
	if changed("slack-username") {
		s.Username = HookSlackUsername
	}
	if changed("slack-icon-url") {
		s.IconURL = HookSlackIconURL
	}
	if changed("slack-color") {
		s.Color = HookSlackColor
	}
	return &s
}

func editOptions(changed func(string) bool) (gol.EditWebhookOptions, *cerr.CustomError) {
	var opts gol.EditWebhookOptions

	if changed("url") {
		opts.URL = gol.Ptr(HookURL)
	}
	if changed("content-type") {
		ct, ce := contentType()
		if ce != nil {
			return opts, ce
		}
		opts.ContentType = gol.Ptr(ct)
	}
	if changed("event") {
		opts.Events = toEvents(HookEvents)
	}
	if changed("branch-filter") {
		opts.BranchFilter = gol.Ptr(HookBranchFilter)
	}
	if changed("active") {
		opts.Active = gol.Ptr(HookActive)
	}
	if changed("insecure-ssl") {
		opts.InsecureSSL = gol.Ptr(HookInsecureSSL)
	}

	secret, set, ce := readSecret(HookSecretFile, HookPromptSecret, "webhook secret")
	if ce != nil {
		return opts, ce
	}
	if set {
		opts.Secret = gol.Ptr(secret)
	}
	header, set, ce := readSecret(HookAuthFile, HookPromptAuth, "authorization header")
	if ce != nil {
		return opts, ce
	}
	if set {
		opts.AuthorizationHeader = gol.Ptr(header)
	}
	return opts, nil
}
