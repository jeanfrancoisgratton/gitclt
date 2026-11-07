// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/issue_mgmt/issues/labels.go
// Original timestamp : 2026.11.06 14:00:00

package issues

import (
	"context"
	"errors"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

var hexColor = regexp.MustCompile(`^[0-9a-f]{6}$`)

// Label is what issue labels shows: a label, and whose it is
type Label struct {
	gol.Label
	// Scope is "repo", or the name of the organization the label belongs to
	Scope string `json:"scope"`
	// Hidden is an organization's label that one of the repository's, of the same name, wins over on issues
	Hidden bool `json:"hidden,omitzero"`
}

// parseColor takes a color with or without its #, in any case; the library wants six lower case digits without it
func parseColor(s string) (string, *cerr.CustomError) {
	color := strings.ToLower(strings.TrimPrefix(s, "#"))
	if !hexColor.MatchString(color) {
		return "", &cerr.CustomError{Title: "Invalid color", Message: s + " is not a color: six hexadecimal digits, such as d73a4a"}
	}
	return color, nil
}

// labelError names what the library refuses about labels
func labelError(title string, e error) *cerr.CustomError {
	switch {
	case errors.Is(e, gol.ErrNameTaken):
		return &cerr.CustomError{Title: "Label already exists", Message: title + ": the name is already in use"}
	case errors.Is(e, gol.ErrAmbiguousName):
		return &cerr.CustomError{Title: "Ambiguous label",
			Message: title + ": several labels have this name; gitclt issue labels shows their IDs, and --id designates one"}
	case gol.IsNotFound(e):
		return &cerr.CustomError{Title: "No such label", Code: http.StatusNotFound, Message: title + ": " + e.Error()}
	}
	return forge.Error(title, e)
}

// orgLabels returns the interface to an organization's labels; only Forgejo has them
func orgLabels(c *forge.Client) (gol.OrgLabels, *cerr.CustomError) {
	l, ok := c.Client.(gol.OrgLabels)
	if !ok {
		return nil, &cerr.CustomError{Title: "Unsupported operation",
			Message: fmt.Sprintf("Connection %s: organization labels only exist on Forgejo", c.Connection.Name)}
	}
	return l, nil
}

func allLabels(ctx context.Context, c *forge.Client, owner, repo string) ([]gol.Label, *cerr.CustomError) {
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Label], error) {
		return c.Labels().List(ctx, owner, repo, o)
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the labels of %s/%s", owner, repo), e)
	}
	return list, nil
}

func allOrgLabels(ctx context.Context, ol gol.OrgLabels, org string) ([]gol.Label, *cerr.CustomError) {
	list, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Label], error) {
		return ol.ListOrgLabels(ctx, org, o)
	})
	if e != nil {
		if gol.IsNotFound(e) {
			return nil, &cerr.CustomError{Title: "No such organization", Code: http.StatusNotFound, Message: org + " is neither an organization nor a user"}
		}
		return nil, forge.Error("Unable to list the labels of "+org, e)
	}
	return list, nil
}

// labelScope is where label commands work: a repository, or with --org an organization; the connection is open
type labelScope struct {
	c           *forge.Client
	owner, repo string
	org         gol.OrgLabels // set with --org
}

func (s labelScope) where() string {
	if s.org != nil {
		return "organization " + LabelOrg
	}
	return s.owner + "/" + s.repo
}

// openLabels connects for a label command: to the repository, or with --org to an organization
func openLabels(ctx context.Context, repo string) (labelScope, *cerr.CustomError) {
	if LabelOrg == "" {
		c, owner, repo, ce := connect(ctx, repo)
		return labelScope{c: c, owner: owner, repo: repo}, ce
	}
	c, ce := forge.Connect(ctx, shared.ConnectionName)
	if ce != nil {
		return labelScope{}, ce
	}
	ol, ce := orgLabels(c)
	if ce != nil {
		return labelScope{}, ce
	}
	return labelScope{c: c, org: ol}, nil
}

// find designates one label of the scope, by name or, with --id, by ID
// Names can repeat on Forgejo: such a name is refused, and the ID then tells them apart
func (s labelScope) find(ctx context.Context, name string, id int64) (*gol.Label, *cerr.CustomError) {
	var list []gol.Label
	var ce *cerr.CustomError
	if s.org != nil {
		list, ce = allOrgLabels(ctx, s.org, LabelOrg)
	} else {
		list, ce = allLabels(ctx, s.c, s.owner, s.repo)
	}
	if ce != nil {
		return nil, ce
	}
	var found []gol.Label
	for _, l := range list {
		if (id != 0 && l.ID == id) || (id == 0 && l.Name == name) {
			found = append(found, l)
		}
	}
	what := "label " + name
	if id != 0 {
		what = fmt.Sprintf("label with ID %d", id)
	}
	switch len(found) {
	case 0:
		return nil, &cerr.CustomError{Title: "No such label", Code: http.StatusNotFound, Message: fmt.Sprintf("%s has no %s", s.where(), what)}
	case 1:
		return &found[0], nil
	}
	ids := make([]string, len(found))
	for i, l := range found {
		ids[i] = strconv.FormatInt(l.ID, 10)
	}
	return nil, &cerr.CustomError{Title: "Ambiguous label",
		Message: fmt.Sprintf("%d labels of %s are named %s; designate one with --id (%s)", len(found), s.where(), name, strings.Join(ids, ", "))}
}

// ListLabelsOf lists the labels an issue of owner/repo can take: the repository's, and on Forgejo its organization's,
// a repository label winning over an organization's of the same name; with --org, only that organization's
func ListLabelsOf(repo string, displayoutput bool) ([]Label, *cerr.CustomError) {
	ctx := context.Background()
	s, ce := openLabels(ctx, repo)
	if ce != nil {
		return nil, ce
	}

	var out []Label
	if s.org != nil {
		list, ce := allOrgLabels(ctx, s.org, LabelOrg)
		if ce != nil {
			return nil, ce
		}
		for _, l := range list {
			out = append(out, Label{Label: l, Scope: LabelOrg})
		}
	} else {
		list, ce := allLabels(ctx, s.c, s.owner, s.repo)
		if ce != nil {
			return nil, ce
		}
		for _, l := range list {
			out = append(out, Label{Label: l, Scope: "repo"})
		}
		// A user owns no labels: the library answers an empty list. The repository's are shown even if the
		// organization's cannot be read
		if ol, ok := s.c.Client.(gol.OrgLabels); ok {
			orgList, ce := allOrgLabels(ctx, ol, s.owner)
			if ce != nil && ce.Code != http.StatusNotFound {
				return nil, ce
			}
			for _, l := range orgList {
				hidden := slices.ContainsFunc(list, func(r gol.Label) bool { return r.Name == l.Name })
				out = append(out, Label{Label: l, Scope: s.owner, Hidden: hidden})
			}
		}
	}

	if !displayoutput {
		return out, nil
	}
	if JSONoutput {
		return out, shared.PrintJSON(out, "label list")
	}
	if shared.QuietOutput {
		// The names an issue can take, once each
		var names []string
		for _, l := range out {
			if !l.Hidden && !slices.Contains(names, l.Name) {
				names = append(names, l.Name)
			}
		}
		for _, n := range names {
			fmt.Println(n)
		}
		return out, nil
	}
	if len(out) == 0 {
		fmt.Printf("%s has no label\n", s.where())
		return out, nil
	}
	renderLabelTable(out)
	return out, nil
}

// GetLabel shows a label, by name or with --id by ID
func GetLabel(repo, name string, displayoutput bool) (*gol.Label, *cerr.CustomError) {
	ctx := context.Background()
	s, ce := openLabels(ctx, repo)
	if ce != nil {
		return nil, ce
	}
	label, ce := s.find(ctx, name, LabelID)
	if ce != nil {
		return nil, ce
	}
	if !displayoutput {
		return label, nil
	}
	if JSONoutput {
		return label, shared.PrintJSON(label, "label")
	}
	renderLabel(label)
	return label, nil
}

// CreateLabel creates a label in owner/repo, or with --org in an organization
func CreateLabel(repo, name string) *cerr.CustomError {
	ctx := context.Background()
	color, ce := parseColor(LabelColor)
	if ce != nil {
		return ce
	}
	s, ce := openLabels(ctx, repo)
	if ce != nil {
		return ce
	}
	want := gol.Label{Name: name, Color: color, Description: LabelDescription}
	var label *gol.Label
	var e error
	if s.org != nil {
		label, e = s.org.CreateOrgLabel(ctx, LabelOrg, want)
	} else {
		label, e = s.c.Labels().Create(ctx, s.owner, s.repo, want)
	}
	if e != nil {
		return labelError(fmt.Sprintf("Unable to create the label %s in %s", name, s.where()), e)
	}
	if JSONoutput {
		return shared.PrintJSON(label, "label")
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Created label %s in %s", hftx.Brown(label.Name), s.where())))
	}
	return nil
}

var labelEditFlags = []string{"name", "color", "description"}

// EditLabel only sends the fields whose flag was explicitly passed; changed is cmd.Flags().Changed
func EditLabel(repo, name string, changed func(string) bool) *cerr.CustomError {
	ctx := context.Background()
	if !slices.ContainsFunc(labelEditFlags, changed) {
		return &cerr.CustomError{Title: "Nothing to edit", Message: "no flag was provided; see gitclt issue label edit --help"}
	}
	var opts gol.EditLabelOptions
	if changed("name") {
		opts.Name = gol.Ptr(LabelName)
	}
	if changed("color") {
		color, ce := parseColor(LabelColor)
		if ce != nil {
			return ce
		}
		opts.Color = gol.Ptr(color)
	}
	if changed("description") {
		opts.Description = gol.Ptr(LabelDescription)
	}

	s, ce := openLabels(ctx, repo)
	if ce != nil {
		return ce
	}
	label, ce := s.find(ctx, name, LabelID)
	if ce != nil {
		return ce
	}
	var e error
	switch {
	case s.org != nil:
		_, e = s.org.EditOrgLabel(ctx, LabelOrg, label.ID, opts)
	case LabelID != 0:
		byID, ce := labelsByID(s.c)
		if ce != nil {
			return ce
		}
		_, e = byID.EditLabelByID(ctx, s.owner, s.repo, label.ID, opts)
	default:
		_, e = s.c.Labels().Edit(ctx, s.owner, s.repo, label.Name, opts)
	}
	if e != nil {
		return labelError("Unable to edit the label "+label.Name, e)
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Updated label %s of %s", hftx.Brown(label.Name), s.where())))
	}
	return nil
}

// labelsByID addresses a repository's labels by ID, to repair names they share; only Forgejo allows such names
func labelsByID(c *forge.Client) (gol.LabelsByID, *cerr.CustomError) {
	l, ok := c.Client.(gol.LabelsByID)
	if !ok {
		return nil, &cerr.CustomError{Title: "Unsupported operation",
			Message: fmt.Sprintf("Connection %s: labels are designated by name on this forge, which refuses repeated names", c.Connection.Name)}
	}
	return l, nil
}

// DeleteLabels deletes labels, after confirmation; every one is checked before asking
// The run stops at the first failure; the labels deleted before it stay deleted
func DeleteLabels(repo string, names []string) *cerr.CustomError {
	ctx := context.Background()
	s, ce := openLabels(ctx, repo)
	if ce != nil {
		return ce
	}

	var targets []*gol.Label
	if LabelID != 0 {
		l, ce := s.find(ctx, "", LabelID)
		if ce != nil {
			return ce
		}
		targets = append(targets, l)
	}
	for _, name := range unique(names) {
		l, ce := s.find(ctx, name, 0)
		if ce != nil {
			return ce
		}
		targets = append(targets, l)
	}
	shown := make([]string, len(targets))
	for i, l := range targets {
		shown[i] = l.Name
	}

	question := fmt.Sprintf("Delete label(s) %s of %s ? They leave every issue carrying them [y/N]: ", strings.Join(shown, ", "), s.where())
	if s.org != nil {
		question = fmt.Sprintf("Delete label(s) %s of %s ? They leave every issue carrying them, in all of its repositories [y/N]: ",
			strings.Join(shown, ", "), s.where())
	}
	if !confirm(question) {
		fmt.Println("Aborted, no label is deleted")
		return nil
	}
	for _, l := range targets {
		var e error
		switch {
		case s.org != nil:
			e = s.org.DeleteOrgLabel(ctx, LabelOrg, l.ID)
		case LabelID != 0:
			byID, ce := labelsByID(s.c)
			if ce != nil {
				return ce
			}
			e = byID.DeleteLabelByID(ctx, s.owner, s.repo, l.ID)
		default:
			e = s.c.Labels().Delete(ctx, s.owner, s.repo, l.Name)
		}
		if e != nil {
			return labelError("Unable to delete the label "+l.Name, e)
		}
		if !shared.QuietOutput {
			fmt.Println(hftx.EnabledSign("Deleted label " + hftx.Brown(l.Name)))
		}
	}
	return nil
}
