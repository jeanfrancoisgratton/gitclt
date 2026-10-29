// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/release_mgmt/tags/list.go
// Original timestamp : 2026.10.29 22:30:00

package tags

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"slices"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
)

// ListTags lists the tags of owner/repo on the server, newest version first, each with its release if it has one
func ListTags(repo string, displayoutput bool) ([]Tag, *cerr.CustomError) {
	ctx := context.Background()
	c, owner, ce := connect(ctx)
	if ce != nil {
		return nil, ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return nil, ce
	}

	all, e := shared.FetchAll(ctx, func(ctx context.Context, o gol.ListOptions) (gol.Page[gol.Tag], error) {
		return c.Tags().List(ctx, owner, repo, o)
	})
	if e != nil {
		return nil, forge.Error(fmt.Sprintf("Unable to list the tags of %s/%s", owner, repo), e)
	}
	byTag, ce := releasesByTag(ctx, c, owner, repo)
	if ce != nil {
		return nil, ce
	}
	list := make([]Tag, 0, len(all))
	for _, t := range all {
		list = append(list, fromServer(t, byTag))
	}
	slices.SortFunc(list, func(a, b Tag) int { return compareVersions(b.Name, a.Name) })

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return list, nil
	}
	if JSONoutput {
		return list, shared.PrintJSON(list, "tag list")
	}
	if len(list) == 0 {
		if !shared.QuietOutput {
			fmt.Printf("%s/%s has no tag\n", owner, repo)
		}
		return list, nil
	}
	renderTagTable(list, c.Backend() != gol.BackendBitbucket)
	return list, nil
}

// GetTag describes one tag of owner/repo on the server: its commit, its annotation, its release and its source archives
func GetTag(repo, tag string, displayoutput bool) (*Tag, *cerr.CustomError) {
	ctx := context.Background()
	c, owner, ce := connect(ctx)
	if ce != nil {
		return nil, ce
	}
	if ce = checkRepo(ctx, c, owner, repo); ce != nil {
		return nil, ce
	}

	st, e := c.Tags().Get(ctx, owner, repo, tag)
	if e != nil {
		if gol.IsNotFound(e) {
			return nil, &cerr.CustomError{Title: "No such tag", Code: http.StatusNotFound,
				Message: fmt.Sprintf("%s/%s has no tag %s", owner, repo, tag)}
		}
		return nil, forge.Error("Unable to get the tag "+tag, e)
	}
	byTag, ce := releasesByTag(ctx, c, owner, repo)
	if ce != nil {
		return nil, ce
	}
	t := fromServer(*st, byTag)

	// we only want the data, we do not need it sent to stdout
	if !displayoutput {
		return &t, nil
	}
	if JSONoutput {
		return &t, shared.PrintJSON(t, "tag info")
	}
	renderTag(t, c.Backend() != gol.BackendBitbucket)
	return &t, nil
}

// fromServer pairs a server tag with its release, if it has one
func fromServer(st gol.Tag, byTag map[string]gol.Release) Tag {
	t := Tag{Name: st.Name, SHA: st.SHA, Message: title(st.Message), TarballURL: st.TarballURL, ZipballURL: st.ZipballURL}
	if r, ok := byTag[st.Name]; ok {
		t.Release, t.ReleaseURL = kind(r), r.HTMLURL
	}
	return t
}
